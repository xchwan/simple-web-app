package booking

import (
	bookingrepo "github.com/xchwan/simple-web-app/internal/booking/repo"
	ticketrepo "github.com/xchwan/simple-web-app/internal/ticket/repo"
	walletrepo "github.com/xchwan/simple-web-app/internal/wallet/repo"
	"gorm.io/gorm"
)

// BookingService 負責訂票相關的業務邏輯。
type BookingService struct {
	db       *bookingrepo.MySQLBookingRepository
	ticketDB *ticketrepo.MySQLTicketRepository
	walletDB *walletrepo.MySQLWalletRepository
	rawDB    *gorm.DB // 用於開啟跨表交易
}

// NewBookingService 建立一個 BookingService。
func NewBookingService(
	db *bookingrepo.MySQLBookingRepository,
	ticketDB *ticketrepo.MySQLTicketRepository,
	walletDB *walletrepo.MySQLWalletRepository,
	rawDB *gorm.DB,
) *BookingService {
	return &BookingService{db: db, ticketDB: ticketDB, walletDB: walletDB, rawDB: rawDB}
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

// Create 建立訂票，在同一個交易內完成：
// 1. 將票券標記為已售出（原子條件 UPDATE）
// 2. 從指定錢包扣款（原子條件 UPDATE）
// 3. 寫入訂票紀錄
func (s *BookingService) Create(callerID, ticketID, walletID int) (*bookingrepo.Booking, error) {
	// 交易前預檢：提早回傳有意義的錯誤，避免不必要的 BEGIN
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

	var created *bookingrepo.Booking
	err := s.rawDB.Transaction(func(tx *gorm.DB) error {
		// 1. 原子性標記售出（並發時的最終防線）
		ok, err := s.ticketDB.WithTx(tx).MarkSold(ticketID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrTicketUnavailable
		}

		// 2. 原子性扣款（並發時的最終防線）
		_, ok, err = s.walletDB.WithTx(tx).Withdraw(walletID, ticket.Price)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInsufficientBalance
		}

		// 3. 寫入訂票紀錄
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
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}
