package infra

import (
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/xchwan/simple-web-app/migrations"
	"gorm.io/gorm"
)

// Migrate 執行所有尚未套用的 migration。
func Migrate(gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}

	driver, err := mysql.WithInstance(sqlDB, &mysql.Config{})
	if err != nil {
		return err
	}

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", src, "mysql", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		// Dirty state 代表上次 migration 執行到一半失敗，version 被標記為 dirty。
		// 搭配 idempotent SQL（CREATE TABLE IF NOT EXISTS），可以安全地 force 清除 dirty
		// 旗標後重新執行，不會有副作用。
		var dirtyErr migrate.ErrDirty
		if errors.As(err, &dirtyErr) {
			if forceErr := m.Force(dirtyErr.Version); forceErr != nil {
				return forceErr
			}
			if upErr := m.Up(); upErr != nil && !errors.Is(upErr, migrate.ErrNoChange) {
				return upErr
			}
			return nil
		}
		return err
	}
	return nil
}
