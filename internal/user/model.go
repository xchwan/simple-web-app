package user

// User 代表系統中的會員。
type User struct {
	ID           int
	Email        string
	Name         string
	PasswordHash string
	Token        string
}
