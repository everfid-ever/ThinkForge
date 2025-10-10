package config

import (
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/elastic/go-elasticsearch/v8"
)

// Config 是系统的全局配置结构体，
// 用于统一管理 Elasticsearch、OpenAI 模型、Embedding 模型等配置信息。
type Config struct {
	Client    *elasticsearch.Client // Elasticsearch 客户端连接实例
	IndexName string                // Elasticsearch 索引名称

	// 以下字段用于 Embedding（向量化）或 Chat 模型
	APIKey         string // OpenAI 或兼容接口的 API Key
	BaseURL        string // 模型服务的 Base URL，可用于自定义代理或私有部署
	EmbeddingModel string // 向量模型名称（用于文本向量化）
	ChatModel      string // 聊天模型名称（用于问答或对话）
}

// GetChatModelConfig 将当前配置转换为 openai.ChatModelConfig 格式，
// 方便在创建 Chat 模型实例时直接使用。
func (x *Config) GetChatModelConfig() *openai.ChatModelConfig {
	if x == nil {
		return nil
	}
	return &openai.ChatModelConfig{
		APIKey:  x.APIKey,    // 模型认证密钥
		BaseURL: x.BaseURL,   // 模型服务地址
		Model:   x.ChatModel, // 使用的 Chat 模型名
	}
}

// Copy 创建当前配置的深拷贝副本，
// 以防止在运行时被修改或造成并发读写问题。
func (x *Config) Copy() *Config {
	return &Config{
		Client:    x.Client,    // 保持相同的 Elasticsearch 客户端引用
		IndexName: x.IndexName, // 复制索引名

		// 以下是 Embedding / Chat 模型的配置复制
		APIKey:         x.APIKey,
		BaseURL:        x.BaseURL,
		EmbeddingModel: x.EmbeddingModel,
		ChatModel:      x.ChatModel,
	}
}
