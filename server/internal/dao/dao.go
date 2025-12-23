package dao

import (
	"context"
	"fmt"
	"time"

	mygorm "github.com/everfid-ever/ThinkForge/internal/model/gorm"
	"github.com/gogf/gf/v2/frame/g"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func init() {
	err := InitDB()
	if err != nil {
		g.Log().Fatal(context.Background(), "database connection not initialized:", err)
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

	// 自动迁移数据库表结构
	if err = mygorm.AutoMigrate(db); err != nil {
		return fmt.Errorf("failed to migrate database tables: %v", err)
	}

	return nil
}

// GetDsn 直接从配置文件读取数据库配置，避免循环依赖
func GetDsn() string {
	ctx := context.Background()

	// 直接从配置文件读取，不要使用 g.DB()
	host := g.Cfg().MustGet(ctx, "database.default.host").String()
	port := g.Cfg().MustGet(ctx, "database.default.port").String()
	user := g.Cfg().MustGet(ctx, "database.default.user").String()
	pass := g.Cfg().MustGet(ctx, "database.default.pass").String()
	name := g.Cfg().MustGet(ctx, "database.default.name").String()
	charset := g.Cfg().MustGet(ctx, "database.default.charset", "utf8mb4").String()

	// 构建 DSN，添加 charset 和其他必要参数
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
		user, pass, host, port, name, charset)
}

func GetDB() *gorm.DB {
	if db == nil {
		g.Log().Fatal(context.Background(), "database connection not initialized")
	}
	return db
}
