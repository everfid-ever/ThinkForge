package gorm

import (
	"github.com/wangle201210/chat-history/models"
	"gorm.io/gorm"
	"time"
)

// KnowledgeBase GORM模型定义
type KnowledgeBase struct {
	ID          int64     `gorm:"primaryKey;column:id"`
	Name        string    `gorm:"column:name;type:varchar(255)"`
	Description string    `gorm:"column:description;type:varchar(255)"`
	Category    string    `gorm:"column:category;type:varchar(255)"`
	Status      int       `gorm:"column:status;default:1"`
	CreateTime  time.Time `gorm:"column:created_at"`
	UpdateTime  time.Time `gorm:"column:updated_at"`
}

// TableName 设置表名
func (KnowledgeBase) TableName() string {
	return "knowledge_base"
}

func autoMigrateTables(db *gorm.DB) error {
	// 自动迁移会创建表、缺失的外键、约束、列和索引
	return db.AutoMigrate(
		&models.Conversation{},
		&models.Message{},
		&models.Attachment{},
		&models.MessageAttachment{},
	)
}
