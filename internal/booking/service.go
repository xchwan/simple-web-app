package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	bookingrepo "github.com/xchwan/simple-web-app/internal/booking/repo"
	ticketrepo "github.com/xchwan/simple-web-app/internal/ticket/repo"
	walletrepo "github.com/xchwan/simple-web-app/internal/wallet/repo"
	"gorm.io/gorm"
)

// bookingMessage 是發往 Kafka 的訊息格式。
type bookingMessage struct {
	BookingID int `json:"bookingId"`
	TicketID  int `json:"ticketId"`
	WalletID  int `json:"walletId"`
	CallerID  int `json:"callerId"`
}

// ticketClaimKey 回傳該張票在 Redis 的搶位 key。
func ticketClaimKey(ticketID int) string {
	return fmt.Sprintf("ticket:claim:%d", ticketID)
}

// BookingService 負責訂票相關的業務邏輯。
type BookingService struct {
	db       *bookingrepo.MySQLBookingRepository
	ticketDB *ticketrepo.MySQLTicketRepository
	walletDB *walletrepo.MySQLWalletRepository
	rawDB    *gorm.DB      // 用於開啟跨表交易
	rdb      *redis.Client // 用於搶票的 Redis 閘門
	kafka    *kafka.Writer // 用於非同步訂票
}

// NewBookingService 建立一個 BookingService。
func NewBookingService(
	db *bookingrepo.MySQLBookingRepository,
	ticketDB *ticketrepo.MySQLTicketRepository,
	walletDB *walletrepo.MySQLWalletRepository,
	rawDB *gorm.DB,
	rdb *redis.Client,
	kafkaWriter *kafka.Writer,
) *BookingService {
	return &BookingService{
		db: db, ticketDB: ticketDB, walletDB: walletDB,
		rawDB: rawDB, rdb: rdb, kafka: kafkaWriter,
	}
}

// GetByID 取得訂票紀錄，僅本人可存取。
func (s *BookingService) GetByID(callerID, bookingID int) (*bookingrepo.Booking, error) {
	b, exists := s.db.FindByID(bookingID)
	if !exists {
		return nil, ErrNotFound
	}
	if b.UserID != callerID {
		return nil, ErrForbidden
	}
	return b, nil
}

// ListByUser 列出指定會員的所有訂票紀錄。
func (s *BookingService) ListByUser(userID int) []*bookingrepo.Booking {
	return s.db.FindByUserID(userID)
}

// Cancel 取消訂票，在同一個交易內完成：
// 1. 將訂票標記為已取消（原子條件 UPDATE）
// 2. 將票券狀態改回 available
// 3. 退款至原錢包
func (s *BookingService) Cancel(callerID, bookingID int) (*bookingrepo.Booking, error) {
	b, exists := s.db.FindByID(bookingID)
	if !exists {
		return nil, ErrNotFound
	}
	if b.UserID != callerID {
		return nil, ErrForbidden
	}
	if b.Status == bookingrepo.StatusCancelled {
		return nil, ErrAlreadyCancelled
	}

	ticket, exists := s.ticketDB.FindByID(b.TicketID)
	if !exists {
		return nil, ErrTicketNotFound
	}

	err := s.rawDB.Transaction(func(tx *gorm.DB) error {
		// 1. 原子性取消（並發時的最終防線）
		ok, err := s.db.WithTx(tx).Cancel(bookingID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrAlreadyCancelled
		}

		// 2. 票券退回 available
		if err := s.ticketDB.WithTx(tx).MarkAvailable(b.TicketID); err != nil {
			return err
		}

		// 3. 退款
		if _, err := s.walletDB.WithTx(tx).Deposit(b.WalletID, ticket.Price); err != nil {
			return err
		}

		b.Status = bookingrepo.StatusCancelled
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 票券退回 available，釋放 Redis 閘門讓票可再次被搶
	s.rdb.Del(context.Background(), ticketClaimKey(b.TicketID))
	return b, nil
}

// Queue 非同步訂票：建立 pending booking 後發 Kafka，立即回傳。
// Consumer 負責後續的 MarkSold + Withdraw + 更新狀態。
func (s *BookingService) Queue(callerID, ticketID, walletID int) (*bookingrepo.Booking, error) {
	ctx := context.Background()

	// Redis 閘門：同一張票只允許一個請求進入流程
	claimed, err := s.rdb.SetNX(ctx, ticketClaimKey(ticketID), callerID, 30*time.Second).Result()
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, ErrTicketUnavailable
	}

	releaseOnFail := true
	defer func() {
		if releaseOnFail {
			s.rdb.Del(ctx, ticketClaimKey(ticketID))
		}
	}()

	// 預檢
	ticket, exists := s.ticketDB.FindByID(ticketID)
	if !exists {
		return nil, ErrTicketNotFound
	}
	if ticket.Status != ticketrepo.StatusAvailable {
		return nil, ErrTicketUnavailable
	}
	wallet, exists := s.walletDB.FindByID(walletID)
	if !exists {
		return nil, ErrWalletNotFound
	}
	if wallet.UserID != callerID {
		return nil, ErrForbidden
	}
	if wallet.Balance < ticket.Price {
		return nil, ErrInsufficientBalance
	}

	// 建立 pending booking（作為 correlation ID）
	b := &bookingrepo.Booking{
		UserID:   callerID,
		TicketID: ticketID,
		WalletID: walletID,
		Status:   bookingrepo.StatusPending,
	}
	if err := s.db.Save(b); err != nil {
		return nil, err
	}

	// kafka 未設定時（測試環境）直接在交易內完成，行為等同 consumer 處理
	if s.kafka == nil {
		if err := s.rawDB.Transaction(func(tx *gorm.DB) error {
			ok, err := s.ticketDB.WithTx(tx).MarkSold(ticketID)
			if err != nil {
				return err
			}
			if !ok {
				return ErrTicketUnavailable
			}
			_, ok, err = s.walletDB.WithTx(tx).Withdraw(walletID, ticket.Price)
			if err != nil {
				return err
			}
			if !ok {
				return ErrInsufficientBalance
			}
			return s.db.WithTx(tx).UpdateStatus(b.ID, bookingrepo.StatusConfirmed)
		}); err != nil {
			s.db.UpdateStatus(b.ID, bookingrepo.StatusFailed)
			return nil, err
		}
		s.rdb.Persist(ctx, ticketClaimKey(ticketID))
		b.Status = bookingrepo.StatusConfirmed
		releaseOnFail = false
		return b, nil
	}

	// 發 Kafka（key=ticketID 確保同票序列化）
	payload, _ := json.Marshal(bookingMessage{
		BookingID: b.ID, TicketID: ticketID, WalletID: walletID, CallerID: callerID,
	})
	if err := s.kafka.WriteMessages(ctx, kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", ticketID)),
		Value: payload,
	}); err != nil {
		s.db.UpdateStatus(b.ID, bookingrepo.StatusFailed)
		return nil, err
	}

	releaseOnFail = false // 後續由 consumer 管理 Redis key
	return b, nil
}

// processBooking 由 Kafka consumer 呼叫，執行實際的扣票 + 扣款交易。
func (s *BookingService) processBooking(msg bookingMessage) {
	ticket, exists := s.ticketDB.FindByID(msg.TicketID)
	if !exists {
		s.fail(msg.BookingID, msg.TicketID)
		return
	}

	err := s.rawDB.Transaction(func(tx *gorm.DB) error {
		ok, err := s.ticketDB.WithTx(tx).MarkSold(msg.TicketID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrTicketUnavailable
		}
		_, ok, err = s.walletDB.WithTx(tx).Withdraw(msg.WalletID, ticket.Price)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInsufficientBalance
		}
		return s.db.WithTx(tx).UpdateStatus(msg.BookingID, bookingrepo.StatusConfirmed)
	})
	if err != nil {
		s.fail(msg.BookingID, msg.TicketID)
		return
	}

	// 訂票成功，Redis key 永久保留
	s.rdb.Persist(context.Background(), ticketClaimKey(msg.TicketID))
	log.Printf("[booking] confirmed bookingID=%d ticketID=%d", msg.BookingID, msg.TicketID)
}

// fail 將訂票標記為失敗並釋放 Redis 閘門。
func (s *BookingService) fail(bookingID, ticketID int) {
	s.db.UpdateStatus(bookingID, bookingrepo.StatusFailed)
	s.rdb.Del(context.Background(), ticketClaimKey(ticketID))
	log.Printf("[booking] failed bookingID=%d ticketID=%d", bookingID, ticketID)
}

