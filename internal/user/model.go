package user

// User 代表系統中的會員。
type User struct {
	ID           int    `gorm:"primarykey;autoIncrement"`
	Email        string `gorm:"uniqueIndex;not null;size:255"`
	Name         string `gorm:"not null;size:255"`
	PasswordHash string `gorm:"not null;size:255"`
	Token        string `gorm:"size:255"`
}
