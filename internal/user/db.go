package user

import (
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// UserRepository 負責會員資料的 MySQL 存取。
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 建立一個 UserRepository。
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Save 新增一位會員，若 email 已存在則回傳 ErrEmailDuplicate。
func (r *UserRepository) Save(u *User) error {
	if err := r.db.Create(u).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return ErrEmailDuplicate
		}
		return err
	}
	return nil
}

// FindByEmailAndPassword 依 email 和密碼 hash 查詢會員。
func (r *UserRepository) FindByEmailAndPassword(email, passwordHash string) (*User, bool) {
	var u User
	if err := r.db.Where("email = ? AND password_hash = ?", email, passwordHash).First(&u).Error; err != nil {
		return nil, false
	}
	return &u, true
}

// FindByID 依會員編號查詢會員。
func (r *UserRepository) FindByID(id int) (*User, bool) {
	var u User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, false
	}
	return &u, true
}

// UpdateName 修改指定會員的名稱。
func (r *UserRepository) UpdateName(id int, newName string) {
	r.db.Model(&User{}).Where("id = ?", id).Update("name", newName)
}

// Search 依關鍵字過濾會員名稱，空字串回傳所有會員。
func (r *UserRepository) Search(keyword string) []*User {
	var users []*User
	query := r.db
	if strings.TrimSpace(keyword) != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	query.Find(&users)
	return users
}
