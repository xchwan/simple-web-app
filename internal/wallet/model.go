package wallet

// Wallet 代表會員的錢包。
type Wallet struct {
	ID      int     `gorm:"primarykey;autoIncrement"`
	UserID  int     `gorm:"not null;index"`
	Name    string  `gorm:"not null;size:255"`
	Balance float64 `gorm:"type:decimal(15,2);not null;default:0"`
}
