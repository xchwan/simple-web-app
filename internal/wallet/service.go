package wallet

// Wallet 代表會員的錢包。
type Wallet struct {
	ID      int     `gorm:"primarykey;autoIncrement"`
	UserID  int     `gorm:"not null;index"`
	Name    string  `gorm:"not null;size:255"`
	Balance float64 `gorm:"type:decimal(15,2);not null;default:0"`
}

// WalletService 負責錢包相關的業務邏輯。
type WalletService struct {
	db *MySQLWalletRepository
}

// NewWalletService 建立一個 WalletService。
func NewWalletService(db *MySQLWalletRepository) *WalletService {
	return &WalletService{db: db}
}

// Create 建立一個新錢包。
func (s *WalletService) Create(userID int, name string) (*Wallet, error) {
	if !validateLength(name, 1, 50) {
		return nil, ErrNameFormatInvalid
	}
	w := &Wallet{UserID: userID, Name: name}
	if err := s.db.Save(w); err != nil {
		return nil, err
	}
	return w, nil
}

// GetByID 取得錢包，僅擁有者可存取。
func (s *WalletService) GetByID(callerID, walletID int) (*Wallet, error) {
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
func (s *WalletService) ListByUser(userID int) []*Wallet {
	return s.db.FindByUserID(userID)
}

func validateLength(s string, min, max int) bool {
	n := len([]rune(s))
	return n >= min && n <= max
}
