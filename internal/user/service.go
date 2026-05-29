package user

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// User 代表系統中的會員。
// Token 不存 DB，改存 Redis（key: session:{token} → userID）。
type User struct {
	ID           int    `gorm:"primarykey;autoIncrement"`
	Email        string `gorm:"uniqueIndex;not null;size:255"`
	Name         string `gorm:"not null;size:255"`
	PasswordHash string `gorm:"not null;size:255"`
}

const (
	sessionPrefix = "session:"
	sessionTTL    = 24 * time.Hour
)

// UserService 負責會員相關的業務邏輯。
type UserService struct {
	repo  *MySQLUserRepository
	redis *redis.Client
}

// NewUserService 建立一個 UserService。
func NewUserService(repo *MySQLUserRepository, rdb *redis.Client) *UserService {
	return &UserService{repo: repo, redis: rdb}
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

// Login 驗證帳密，產生 token 存入 Redis 並回傳。
func (s *UserService) Login(email, password string) (*User, string, error) {
	if !validateEmail(email) || !validateLength(password, 5, 32) {
		return nil, "", ErrLoginFormatInvalid
	}
	u, exists := s.repo.FindByEmailAndPassword(email, hashPassword(password))
	if !exists {
		return nil, "", ErrCredentialsInvalid
	}
	token := uuid.New().String()
	if err := s.redis.Set(context.Background(), sessionPrefix+token, u.ID, sessionTTL).Err(); err != nil {
		return nil, "", err
	}
	return u, token, nil
}

// Authenticate 從 Redis 驗證 token，回傳對應的會員。
func (s *UserService) Authenticate(token string) (*User, error) {
	if token == "" {
		return nil, ErrTokenInvalid
	}
	val, err := s.redis.Get(context.Background(), sessionPrefix+token).Result()
	if err != nil {
		return nil, ErrTokenInvalid
	}
	userID, err := strconv.Atoi(val)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	u, exists := s.repo.FindByID(userID)
	if !exists {
		return nil, ErrTokenInvalid
	}
	return u, nil
}

// Logout 從 Redis 刪除 session token。
func (s *UserService) Logout(token string) {
	s.redis.Del(context.Background(), sessionPrefix+token)
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
