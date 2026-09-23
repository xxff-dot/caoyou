package store

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(driver, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case "sqlite", "":
		dialector = sqlite.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", driver)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Mailbox{}, &Message{}, &Attachment{}, &Setting{})
}

// GetSetting / SetSetting 通用KV
func GetSetting(db *gorm.DB, key string) (string, bool) {
	var s Setting
	if err := db.First(&s, "key = ?", key).Error; err != nil {
		return "", false
	}
	return s.Value, true
}

func SetSetting(db *gorm.DB, key, value string) {
	db.Save(&Setting{Key: key, Value: value})
}
