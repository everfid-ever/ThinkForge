package rag

import (
	"context"

	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/internal/logic/chat"
)

// Chat 处理 RAG 聊天请求（/v1/chat）。
// 它负责：
//  1. 调用 Retriever 检索相关文档；
//  2. 调用 Chat 逻辑模块生成最终回答；
//  3. 将回答封装到 ChatRes 并返回。
func (c *ControllerV1) Chat(ctx context.Context, req *v1.ChatReq) (res *v1.ChatRes, err error) {
	// Step 1: 调用 Retriever 获取相关文档。
	// 构造一个 RetrieverReq 请求，将用户的问题、TopK、Score 等参数传入。
	retriever, err := c.Retriever(ctx, &v1.RetrieverReq{
		Question: req.Question,
		TopK:     req.TopK,
		Score:    req.Score,
	})
	if err != nil {
		// 如果文档检索失败，直接返回错误。
		return
	}

	// Step 2: 获取聊天模块实例。
	// chat.GetChat() 通常返回一个全局或单例的 Chat 引擎对象，
	// 它内部封装了与大语言模型 (LLM) 交互的逻辑，例如调用 OpenAI、Claude 或本地模型。
	chatI := chat.GetChat()

	// Step 3: 生成回答。
	// 通过 chatI.GetAnswer 调用模型生成回答，传入参数包括：
	//   - ctx: 上下文对象（用于超时/取消控制）
	//   - req.ConvID: 会话ID（用于多轮对话上下文管理）
	//   - retriever.Document: 检索得到的相关文档
	//   - req.Question: 用户的问题内容
	answer, err := chatI.GetAnswer(ctx, req.ConvID, retriever.Document, req.Question)
	if err != nil {
		// 如果生成回答失败（例如 LLM 调用出错），返回错误。
		return
	}

	// Step 4: 构造响应结果并返回。
	res = &v1.ChatRes{
		Answer: answer, // 模型生成的回答文本
	}
	return
}
