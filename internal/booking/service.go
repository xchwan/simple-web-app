package booking

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	bookingrepo "github.com/xchwan/simple-web-app/internal/booking/repo"
	ticketrepo "github.com/xchwan/simple-web-app/internal/ticket/repo"
	walletrepo "github.com/xchwan/simple-web-app/internal/wallet/repo"
	"gorm.io/gorm"
)

// ticketClaimKey 回傳該張票在 Redis 的搶位 key。
func ticketClaimKey(ticketID int) string {
	return fmt.Sprintf("ticket:claim:%d", ticketID)
}

// BookingService 負責訂票相關的業務邏輯。
type BookingService struct {
	db       *bookingrepo.MySQLBookingRepository
	ticketDB *ticketrepo.MySQLTicketRepository
	walletDB *walletrepo.MySQLWalletRepository
	rawDB    *gorm.DB        // 用於開啟跨表交易
	rdb      *redis.Client   // 用於搶票的 Redis 閘門
}

// NewBookingService 建立一個 BookingService。
func NewBookingService(
	db *bookingrepo.MySQLBookingRepository,
	ticketDB *ticketrepo.MySQLTicketRepository,
	walletDB *walletrepo.MySQLWalletRepository,
	rawDB *gorm.DB,
	rdb *redis.Client,
) *BookingService {
	return &BookingService{db: db, ticketDB: ticketDB, walletDB: walletDB, rawDB: rawDB, rdb: rdb}
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

// Create 建立訂票。
// Redis 閘門：SETNX 確保同一張票只有一個請求進到 MySQL，其餘直接回 409。
// MySQL 交易：MarkSold + Withdraw + Save，任一步驟失敗整體 rollback。
func (s *BookingService) Create(callerID, ticketID, walletID int) (*bookingrepo.Booking, error) {
	// ── Redis 閘門（第一道防線，保護 MySQL）──────────────────────────────
	ctx := context.Background()
	claimed, err := s.rdb.SetNX(ctx, ticketClaimKey(ticketID), callerID, 30*time.Second).Result()
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, ErrTicketUnavailable // 其他人已搶到，直接拒絕
	}

	// 若後續流程失敗，釋放 Redis 鎖讓票券可再次被搶
	releaseOnFail := true
	defer func() {
		if releaseOnFail {
			s.rdb.Del(ctx, ticketClaimKey(ticketID))
		}
	}()

	// ── 交易前預檢（快速回傳有意義的錯誤）──────────────────────────────
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

	// ── MySQL 交易（第二道防線，保證原子性）─────────────────────────────
	var created *bookingrepo.Booking
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

		b := &bookingrepo.Booking{
			UserID:   callerID,
			TicketID: ticketID,
			WalletID: walletID,
			Status:   bookingrepo.StatusConfirmed,
		}
		if err := s.db.WithTx(tx).Save(b); err != nil {
			return err
		}
		created = b
		return nil
	}); err != nil {
		return nil, err
	}

	releaseOnFail = false // 訂票成功，Redis key 永久保留（票已售出）
	return created, nil
}
