package database

import (
	"fmt"
	"log"
	"time"

	"meta-file-system/conf"
	"meta-file-system/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// UploaderDB global GORM database instance for Uploader service (always MySQL)
var UploaderDB *gorm.DB

// InitUploaderDB initialize Uploader database (always MySQL)
func InitUploaderDB() error {
	var err error

	// Build DSN
	dsn := conf.Cfg.Database.Dsn
	if dsn == "" {
		return fmt.Errorf("database dsn is empty")
	}

	// Connect database
	UploaderDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	// Get underlying sql.DB
	sqlDB, err := UploaderDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Set connection pool
	sqlDB.SetMaxOpenConns(conf.Cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(conf.Cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Uploader database (MySQL) connected successfully")
	return nil
}

// AutoMigrate auto migrate database table structure for Uploader
func AutoMigrate() error {
	return UploaderDB.AutoMigrate(
		&model.File{},
		&model.FileChunk{},
		&model.Assistant{},
	)
}

// CloseUploaderDB close Uploader database connection
func CloseUploaderDB() error {
	if UploaderDB == nil {
		return nil
	}
	sqlDB, err := UploaderDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
