package wallet

import "gorm.io/gorm"

// WalletDB 負責錢包資料的 MySQL 存取。
type WalletDB struct {
	db *gorm.DB
}

// NewWalletDB 建立一個 WalletDB。
func NewWalletDB(db *gorm.DB) *WalletDB {
	return &WalletDB{db: db}
}

// Save 新增一個錢包。
func (r *WalletDB) Save(w *Wallet) error {
	return r.db.Create(w).Error
}

// FindByID 依錢包編號查詢。
func (r *WalletDB) FindByID(id int) (*Wallet, bool) {
	var w Wallet
	if err := r.db.First(&w, id).Error; err != nil {
		return nil, false
	}
	return &w, true
}

// FindByUserID 查詢指定會員的所有錢包。
func (r *WalletDB) FindByUserID(userID int) []*Wallet {
	var wallets []*Wallet
	r.db.Where("user_id = ?", userID).Find(&wallets)
	return wallets
}
