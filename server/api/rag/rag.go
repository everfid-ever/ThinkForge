package rag

import (
	"context"

	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
)

// IRagV1 定义了 RAG（Retrieval-Augmented Generation）模块的接口规范。
//
// 该接口统一定义了智能问答系统的核心能力：
//   - Chat：基于知识增强生成（RAG）的对话问答。
//   - Retriever：基于语义相似度的文档检索功能。
//   - Indexer：文档入库（索引）功能，用于构建或更新知识库。
//
// 实现该接口的控制器（ControllerV1）通常负责：
//   - 接收 HTTP 请求并验证参数；
//   - 调用底层逻辑服务（如 core.Rag 或 chat 模块）；
//   - 返回结构化的响应结果。
type IRagV1 interface {

	// Chat 处理基于知识增强的聊天请求。
	//
	// 功能说明：
	//   - 根据用户输入问题，从知识库中检索相关文档；
	//   - 结合检索结果生成上下文增强的回答；
	//   - 支持多轮对话（会话ID控制）。
	//
	// 参数：
	//   - ctx: 上下文对象，用于控制请求的生命周期。
	//   - req: ChatReq，请求体，包含问题、会话ID、TopK、Score 等参数。
	//
	// 返回值：
	//   - res: ChatRes，响应体，包含模型生成的答案。
	//   - err: 错误信息（如果执行失败）。
	Chat(ctx context.Context, req *v1.ChatReq) (res *v1.ChatRes, err error)

	// Retriever 处理文档检索请求。
	//
	// 功能说明：
	//   - 根据问题（Question）在知识库中检索最相关的文档；
	//   - 返回排序后的文档及其元数据；
	//   - 可指定 TopK（检索数量）和 Score（相似度阈值）。
	//
	// 参数：
	//   - ctx: 上下文对象。
	//   - req: RetrieverReq，请求体，包含问题、TopK、Score、知识库等信息。
	//
	// 返回值：
	//   - res: RetrieverRes，包含检索到的文档列表。
	//   - err: 错误信息（如果执行失败）。
	Retriever(ctx context.Context, req *v1.RetrieverReq) (res *v1.RetrieverRes, err error)

	// Indexer 处理文档入库（索引）请求。
	//
	// 功能说明：
	//   - 支持本地文件上传或网络URL文件入库；
	//   - 将文档解析、分块并生成向量后存入知识库；
	//   - 返回所有被成功索引的文档ID。
	//
	// 参数：
	//   - ctx: 上下文对象。
	//   - req: IndexerReq，请求体，包含文件、URL 和知识库名称。
	//
	// 返回值：
	//   - res: IndexerRes，响应体，包含索引生成的文档ID列表。
	//   - err: 错误信息（如果执行失败）。
	Indexer(ctx context.Context, req *v1.IndexerReq) (res *v1.IndexerRes, err error)
}
