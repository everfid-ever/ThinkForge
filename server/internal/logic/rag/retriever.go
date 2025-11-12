package rag

import (
	"github.com/elastic/go-elasticsearch/v8"         // ElasticSearch v8 官方客户端
	"github.com/everfid-ever/ThinkForge/core"        // RAG 核心逻辑封装
	"github.com/everfid-ever/ThinkForge/core/config" // RAG 配置结构定义
	"github.com/gogf/gf/v2/frame/g"                  // GoFrame 配置与日志模块
	"github.com/gogf/gf/v2/os/gctx"
)

//
// ===================== 全局变量定义 =====================
//

// ragSvr 是包级全局变量，代表 RAG 核心服务实例（负责向量检索与生成）
// 它在 init() 初始化时创建，并通过 GetRagSvr() 提供给外部模块使用。
var ragSvr = &core.Rag{}

//
// ===================== 初始化逻辑 =====================
//

// init() 在包导入时自动执行。
// 功能：
// 1. 读取配置文件中的 ElasticSearch 与 Embedding 模型配置。
// 2. 创建 ES 客户端。
// 3. 创建 core.Rag 实例（封装向量检索与语义嵌入功能）。
// 4. 若任意步骤失败，则打印错误日志并停止后续初始化。
func init() {
	ctx := gctx.New()

	// Step 1: 创建 ElasticSearch 客户端
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Username: g.Cfg().MustGet(ctx, "es.username").String(),
		Password: g.Cfg().MustGet(ctx, "es.password").String(),
		Addresses: []string{
			g.Cfg().MustGet(ctx, "es.address").String(), // 从配置读取 ES 地址
		},
	})
	if err != nil {
		g.Log().Fatalf(ctx, "NewClient of es8 failed, err=%v", err)
		return
	}

	// Step 2: 初始化 core.Rag 服务
	ragSvr, err = core.New(ctx, &config.Config{
		Client:    client,                                             // ES 客户端实例
		IndexName: g.Cfg().MustGet(ctx, "es.indexName").String(),      // 索引名（如 thinkforge_docs）
		APIKey:    g.Cfg().MustGet(ctx, "embedding.apiKey").String(),  // 向量嵌入 API Key
		BaseURL:   g.Cfg().MustGet(ctx, "embedding.baseURL").String(), // 向量 API Base URL
		ChatModel: g.Cfg().MustGet(ctx, "embedding.model").String(),   // 嵌入模型名（如 text-embedding-3-small）
	})
	if err != nil {
		g.Log().Fatalf(ctx, "New of rag failed, err=%v", err)
		return
	}
}

//
// ===================== 外部访问接口 =====================
//

// GetRagSvr 返回全局 RAG 服务实例。
// 外部模块（如 ControllerV1）通过此函数获取 RAG 服务对象，用于执行文档检索（Retrieve）等操作。
func GetRagSvr() *core.Rag {
	return ragSvr
}
