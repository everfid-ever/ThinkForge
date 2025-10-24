package rag

import (
	"context"

	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
)

// IRagV1 定义了 RAG（Retrieval-Augmented Generation）模块的接口规范。
// 该接口统一定义了聊天与文档检索的核心服务方法。
// 实现该接口的结构体通常负责处理具体的业务逻辑，例如调用大模型或文档检索引擎。
type IRagV1 interface {

	// Chat 处理聊天请求。
	// 参数：
	//   - ctx: 上下文对象，用于控制请求生命周期、超时和取消。
	//   - req: ChatReq 请求体，包含会话ID、问题内容及可选检索参数。
	// 返回值：
	//   - res: ChatRes 响应体，包含生成的答案。
	//   - err: 错误信息（如果执行失败）。
	Chat(ctx context.Context, req *v1.ChatReq) (res *v1.ChatRes, err error)

	// Retriever 处理文档检索请求。
	// 参数：
	//   - ctx: 上下文对象。
	//   - req: RetrieverReq 请求体，包含问题、知识库名称、TopK、Score 等参数。
	// 返回值：
	//   - res: RetrieverRes 响应体，包含检索到的文档列表。
	//   - err: 错误信息（如果执行失败）。
	Retriever(ctx context.Context, req *v1.RetrieverReq) (res *v1.RetrieverRes, err error)
}
