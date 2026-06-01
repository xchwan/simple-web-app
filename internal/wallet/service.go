package wallet

import walletrepo "github.com/xchwan/simple-web-app/internal/wallet/repo"

// WalletService 負責錢包相關的業務邏輯。
type WalletService struct {
	db *walletrepo.MySQLWalletRepository
}

// NewWalletService 建立一個 WalletService。
func NewWalletService(db *walletrepo.MySQLWalletRepository) *WalletService {
	return &WalletService{db: db}
}

// Create 建立一個新錢包。
func (s *WalletService) Create(userID int, name string) (*walletrepo.Wallet, error) {
	if !validateLength(name, 1, 50) {
		return nil, ErrNameFormatInvalid
	}
	w := &walletrepo.Wallet{UserID: userID, Name: name}
	if err := s.db.Save(w); err != nil {
		return nil, err
	}
	return w, nil
}

// GetByID 取得錢包，僅擁有者可存取。
func (s *WalletService) GetByID(callerID, walletID int) (*walletrepo.Wallet, error) {
	w, exists := s.db.FindByID(walletID)
	if !exists {
		return nil, ErrNotFound
	}
	if w.UserID != callerID {
		return nil, ErrForbidden
	}
	return w, nil
}

// ListByUser 列出指定會員的所有錢包。
func (s *WalletService) ListByUser(userID int) []*walletrepo.Wallet {
	return s.db.FindByUserID(userID)
}

// Deposit 存款，僅擁有者可操作，amount 必須 > 0。
func (s *WalletService) Deposit(callerID, walletID int, amount float64) (*walletrepo.Wallet, error) {
	if amount <= 0 {
		return nil, ErrAmountInvalid
	}
	w, exists := s.db.FindByID(walletID)
	if !exists {
		return nil, ErrNotFound
	}
	if w.UserID != callerID {
		return nil, ErrForbidden
	}
	return s.db.Deposit(walletID, amount)
}

// Withdraw 提款，僅擁有者可操作，amount 必須 > 0 且餘額充足。
func (s *WalletService) Withdraw(callerID, walletID int, amount float64) (*walletrepo.Wallet, error) {
	if amount <= 0 {
		return nil, ErrAmountInvalid
	}
	w, exists := s.db.FindByID(walletID)
	if !exists {
		return nil, ErrNotFound
	}
	if w.UserID != callerID {
		return nil, ErrForbidden
	}
	updated, ok, err := s.db.Withdraw(walletID, amount)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInsufficientBalance
	}
	return updated, nil
}

func validateLength(s string, min, max int) bool {
	n := len([]rune(s))
	return n >= min && n <= max
}
