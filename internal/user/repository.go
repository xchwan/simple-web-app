package user

import (
	"strings"
	"sync"
)

// UserRepository 負責會員資料的記憶體存取。
type UserRepository struct {
	mu     sync.RWMutex
	users  []*User
	nextID int
}

// NewUserRepository 建立一個空的 UserRepository。
func NewUserRepository() *UserRepository {
	return &UserRepository{nextID: 1}
}

// Save 新增一位會員，若 email 已存在則回傳 ErrEmailDuplicate。
func (r *UserRepository) Save(u *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if existing.Email == u.Email {
			return ErrEmailDuplicate
		}
	}
	u.ID = r.nextID
	r.nextID++
	r.users = append(r.users, u)
	return nil
}

// FindByEmailAndPassword 依 email 和密碼 hash 查詢會員。
func (r *UserRepository) FindByEmailAndPassword(email, passwordHash string) (*User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.Email == email && u.PasswordHash == passwordHash {
			return u, true
		}
	}
	return nil, false
}

// FindByToken 依 token 查詢會員。
func (r *UserRepository) FindByToken(token string) (*User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.Token == token {
			return u, true
		}
	}
	return nil, false
}

// FindByID 依會員編號查詢會員。
func (r *UserRepository) FindByID(id int) (*User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.ID == id {
			return u, true
		}
	}
	return nil, false
}

// UpdateName 修改指定會員的名稱。
func (r *UserRepository) UpdateName(id int, newName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.ID == id {
			u.Name = newName
			return
		}
	}
}

// Search 依關鍵字過濾會員名稱，空字串回傳所有會員。
func (r *UserRepository) Search(keyword string) []*User {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if keyword == "" {
		result := make([]*User, len(r.users))
		copy(result, r.users)
		return result
	}
	var result []*User
	for _, u := range r.users {
		if strings.Contains(u.Name, keyword) {
			result = append(result, u)
		}
	}
	return result
}
