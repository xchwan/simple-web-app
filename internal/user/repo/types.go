package userrepo

import "errors"

// User 代表系統中的會員。
// Token 不存 DB，改存 Redis（key: session:{token} → userID）。
type User struct {
	ID           int    `gorm:"primarykey;autoIncrement"`
	Email        string `gorm:"uniqueIndex;not null;size:255"`
	Name         string `gorm:"not null;size:255"`
	PasswordHash string `gorm:"not null;size:255"`
}

// ErrEmailDuplicate 由 MySQL unique constraint 觸發，表示 email 已被使用。
var ErrEmailDuplicate = errors.New("email duplicate")
