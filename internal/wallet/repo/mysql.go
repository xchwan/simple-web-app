package walletrepo

import "gorm.io/gorm"

// MySQLWalletRepository 以 MySQL 實作 WalletRepository。
type MySQLWalletRepository struct {
	db *gorm.DB
}

// NewMySQLWalletRepository 建立一個 MySQLWalletRepository。
func NewMySQLWalletRepository(db *gorm.DB) *MySQLWalletRepository {
	return &MySQLWalletRepository{db: db}
}

// Save 新增一個錢包。
func (r *MySQLWalletRepository) Save(w *Wallet) error {
	return r.db.Create(w).Error
}

// FindByID 依錢包編號查詢。
func (r *MySQLWalletRepository) FindByID(id int) (*Wallet, bool) {
	var w Wallet
	if err := r.db.First(&w, id).Error; err != nil {
		return nil, false
	}
	return &w, true
}

// FindByUserID 查詢指定會員的所有錢包。
func (r *MySQLWalletRepository) FindByUserID(userID int) []*Wallet {
	var wallets []*Wallet
	r.db.Where("user_id = ?", userID).Find(&wallets)
	return wallets
}
