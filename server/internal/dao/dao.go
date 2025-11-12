package dao

import (
	"context"
	"fmt"
	mygorm "github.com/everfid-ever/ThinkForge/internal/model/gorm"
	"gorm.io/driver/mysql"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func init() {
	err := InitDB()
	if err != nil {
		g.Log().Fatal(context.Background(), "database connection not initialized:")
	}
}

func InitDB() error {
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}
	dsn := GetDsn()

	var err error
	db, err = gorm.Open(mysql.Open(dsn), config)
	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err = autoMigrateTables(); err != nil {
		return fmt.Errorf("auto migrate tables failed: %v", err)
	}
	return nil
}

func GetDsn() string {
	cfg := g.DB().GetConfig()
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", cfg.User, cfg.Pass, cfg.Host, cfg.Port, cfg.Name)
}

func GetDB() *gorm.DB {
	if db == nil {
		g.Log().Fatal(context.Background(), "database connection not initialized")
	}
	return db
}

func autoMigrateTables() error {
	return db.AutoMigrate(&mygorm.KnowledgeBase{})
}
