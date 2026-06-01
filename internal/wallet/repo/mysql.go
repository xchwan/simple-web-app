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

// WithTx 回傳一個綁定到 tx 的新 repository 實例，供跨表交易使用。
func (r *MySQLWalletRepository) WithTx(tx *gorm.DB) *MySQLWalletRepository {
	return &MySQLWalletRepository{db: tx}
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

// Deposit 原子性地將 amount 加到餘額，回傳更新後的錢包。
func (r *MySQLWalletRepository) Deposit(id int, amount float64) (*Wallet, error) {
	if err := r.db.Exec("UPDATE wallets SET balance = balance + ? WHERE id = ?", amount, id).Error; err != nil {
		return nil, err
	}
	w, _ := r.FindByID(id)
	return w, nil
}

// Withdraw 原子性地扣款：只有 balance >= amount 才會執行。
// ok=false 表示餘額不足（非 DB 錯誤）。
func (r *MySQLWalletRepository) Withdraw(id int, amount float64) (w *Wallet, ok bool, err error) {
	result := r.db.Exec(
		"UPDATE wallets SET balance = balance - ? WHERE id = ? AND balance >= ?",
		amount, id, amount,
	)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, false, nil
	}
	w, _ = r.FindByID(id)
	return w, true, nil
}
