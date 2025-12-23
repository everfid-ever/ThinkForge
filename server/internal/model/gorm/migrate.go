package gorm

import (
	"fmt"
	"gorm.io/gorm"
)

// AutoMigrate 自动迁移所有GORM模型
func AutoMigrate(db *gorm.DB) error {
	fmt.Println("Start to migrate KnowledgeBase...")
	if err := db.AutoMigrate(&KnowledgeBase{}); err != nil {
		return fmt.Errorf("KnowledgeBase migration is failed: %v", err)
	}
	fmt.Println("✓ KnowledgeBase migration is successful")

	fmt.Println("Start to migrate KnowledgeDocuments...")
	if err := db.AutoMigrate(&KnowledgeDocuments{}); err != nil {
		return fmt.Errorf("KnowledgeDocuments migration is failed: %v", err)
	}
	fmt.Println("✓ KnowledgeDocuments migration is successful")

	fmt.Println("Start to migrate KnowledgeChunks...")
	if err := db.AutoMigrate(&KnowledgeChunks{}); err != nil {
		return fmt.Errorf("KnowledgeChunks migration is failed: %v", err)
	}
	fmt.Println("✓ KnowledgeChunks migration is successful ")

	return nil
}
