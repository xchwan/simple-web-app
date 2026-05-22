package user

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// UserService 負責會員相關的業務邏輯。
type UserService struct {
	repo *UserRepository
}

// NewUserService 建立一個 UserService。
func NewUserService(repo *UserRepository) *UserService {
	return &UserService{repo: repo}
}

// Register 驗證並新增一位會員。
func (s *UserService) Register(email, name, password string) (*User, error) {
	if !validateEmail(email) || !validateLength(name, 5, 32) || !validateLength(password, 5, 32) {
		return nil, ErrRegisterFormatInvalid
	}
	u := &User{
		Email:        email,
		Name:         name,
		PasswordHash: hashPassword(password),
	}
	if err := s.repo.Save(u); err != nil {
		return nil, err
	}
	return u, nil
}

// Login 驗證帳密並產生 token。
func (s *UserService) Login(email, password string) (*User, error) {
	if !validateEmail(email) || !validateLength(password, 5, 32) {
		return nil, ErrLoginFormatInvalid
	}
	u, exists := s.repo.FindByEmailAndPassword(email, hashPassword(password))
	if !exists {
		return nil, ErrCredentialsInvalid
	}
	u.Token = uuid.New().String()
	return u, nil
}

// Authenticate 驗證 token 並回傳對應的會員。
func (s *UserService) Authenticate(token string) (*User, error) {
	if !validateLength(token, 36, 60) {
		return nil, ErrTokenInvalid
	}
	u, exists := s.repo.FindByToken(token)
	if !exists {
		return nil, ErrTokenInvalid
	}
	return u, nil
}

// UpdateName 驗證身份並修改會員名稱。
func (s *UserService) UpdateName(callerID, targetID int, newName string) error {
	if callerID != targetID {
		return ErrForbidden
	}
	if !validateLength(newName, 5, 32) {
		return ErrNameFormatInvalid
	}
	s.repo.UpdateName(targetID, newName)
	return nil
}

// SearchUsers 依關鍵字查詢會員列表。
func (s *UserService) SearchUsers(keyword string) []*User {
	return s.repo.Search(keyword)
}

// ===== 私有輔助函式 =====

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", sum)
}

func validateEmail(email string) bool {
	return validateLength(email, 4, 32) && strings.Contains(email, "@")
}

func validateLength(s string, min, max int) bool {
	n := len([]rune(s))
	return n >= min && n <= max
}
