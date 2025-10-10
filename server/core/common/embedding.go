package common

import (
	"context"
	"github.com/everfid-ever/ThinkForge/core/config"
	"os"

	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/embedding"
)

// NewEmbedding 创建并初始化一个基于 OpenAI 的向量嵌入器 (Embedder)。
// 该函数用于在 RAG（Retrieval-Augmented Generation）系统中生成文本向量表示。
//
// 参数：
//   - ctx: 上下文对象，用于控制请求生命周期、超时或取消操作。
//   - conf: 配置结构体，包含 API Key、模型名称、BaseURL 等信息。
//
// 返回值：
//   - eb: 一个实现 embedding.Embedder 接口的实例（可直接用于文本嵌入计算）。
//   - err: 创建过程中出现的错误（如配置错误或网络异常）。
//
// 逻辑说明：
//  1. 从 config.Config 中读取嵌入配置（API Key、模型名称、BaseURL）。
//  2. 若配置项缺失，则从环境变量读取（OPENAI_API_KEY、OPENAI_BASE_URL）。
//  3. 若模型名称未指定，默认使用 "text-embedding-3-large"。
//  4. 调用 openai.NewEmbedder() 创建 Embedding 客户端。
//  5. 返回可直接使用的 embedding.Embedder 实例。
func NewEmbedding(ctx context.Context, conf *config.Config) (eb embedding.Embedder, err error) {
	// 初始化 embedding 模型配置
	econf := &openai.EmbeddingConfig{
		APIKey:     conf.APIKey,         // OpenAI API Key（模型访问凭证）
		Model:      conf.EmbeddingModel, // 向量化模型名称
		Dimensions: Of(1024),            // 向量维度（默认 1024 维，可根据模型不同调整）
		Timeout:    0,                   // 超时时间，0 表示不限制
		BaseURL:    conf.BaseURL,        // 模型服务地址（可为自定义代理或本地部署）
	}

	// 若配置中未指定 API Key，则尝试从环境变量读取
	if econf.APIKey == "" {
		econf.APIKey = os.Getenv("OPENAI_API_KEY")
	}

	// 若配置中未指定 BaseURL，则尝试从环境变量读取
	if econf.BaseURL == "" {
		econf.BaseURL = os.Getenv("OPENAI_BASE_URL")
	}

	// 若未指定模型名称，则使用默认 embedding 模型
	if econf.Model == "" {
		econf.Model = "text-embedding-3-large"
	}

	// 使用配置创建 OpenAI Embedding 实例
	eb, err = openai.NewEmbedder(ctx, econf)
	if err != nil {
		return nil, err
	}

	// 返回创建好的向量化模型实例
	return eb, nil
}
