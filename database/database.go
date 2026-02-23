package database

import (
	"fmt"
	"go-music-tag/config"
	"go-music-tag/models"
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/mattn/go-sqlite3"
)

var DB *gorm.DB

func Init() error {
	cfg := config.GetConfig()

	dbDir := filepath.Dir(cfg.Database.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(cfg.Database.Path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := DB.AutoMigrate(&models.Music{}, &models.ScanLog{}, &models.WebDAVConfig{}); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	initDefaultWebDAVConfig(cfg)

	log.Println("Database initialized successfully")
	return nil
}

func initDefaultWebDAVConfig(cfg *config.Config) {
	var count int64
	DB.Model(&models.WebDAVConfig{}).Count(&count)

	if count == 0 {
		defaultConfig := &models.WebDAVConfig{
			URL:        cfg.WebDAV.URL,
			Username:   cfg.WebDAV.Username,
			Password:   cfg.WebDAV.Password,
			RootPath:   cfg.WebDAV.RootPath,
			Enabled:    true,
			TestStatus: "pending",
		}
		DB.Create(defaultConfig)
		log.Println("Default WebDAV config created")
	}
}

func GetDB() *gorm.DB {
	return DB
}
