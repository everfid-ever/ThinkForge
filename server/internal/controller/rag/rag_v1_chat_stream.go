package rag

import (
	"context"
	"io"

	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/core/agent"
	"github.com/everfid-ever/ThinkForge/internal/logic/chat"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

// ChatStream 流式对话接口（支持 Agentic 模式）
func (c *ControllerV1) ChatStream(ctx context.Context, req *v1.ChatStreamReq) (res *v1.ChatStreamRes, err error) {
	g.Log().Infof(ctx, "🚀 Stream RAG: %s", req.Question)

	// ===== 判断是否启用智能模式 =====
	if !req.EnableAgentic {
		// 传统流式 RAG
		return c.legacyStreamRAG(ctx, req)
	}

	// ===== Agentic 流式 RAG =====

	// Step 1: 意图识别（快速，不阻塞流式）
	classifier := agent.NewHybridIntentClassifierRuleOnly() // 仅用规则，保证快速
	intent, err := classifier.Classify(ctx, req.Question)
	if err != nil {
		g.Log().Warningf(ctx, "Intent classification failed: %v, fallback to legacy", err)
		return c.legacyStreamRAG(ctx, req)
	}

	g.Log().Debugf(ctx, "Intent: type=%s, strategy=%s", intent.Type, intent.Strategy)

	// Step 2: 简单问题 → 直接流式返回
	if intent.Complexity == agent.ComplexitySimple {
		return c.legacyStreamRAG(ctx, req)
	}

	// Step 3: 复杂问题 → 流式返回推理步骤 + 答案
	// TODO: 实现流式 ReAct Agent
	// 当前降级到传统流式
	return c.legacyStreamRAG(ctx, req)
}

// legacyStreamRAG 传统流式 RAG（保留原逻辑）
func (c *ControllerV1) legacyStreamRAG(ctx context.Context, req *v1.ChatStreamReq) (res *v1.ChatStreamRes, err error) {
	// Step 1: 检索
	retriever, err := c.Retriever(ctx, &v1.RetrieverReq{
		Question:      req.Question,
		TopK:          req.TopK,
		Score:         req.Score,
		KnowledgeName: req.KnowledgeName,
	})
	if err != nil {
		g.Log().Error(ctx, "Retriever failed:", err)
		return nil, err
	}

	// Step 2: 获取 Chat 实例
	chatI := chat.GetChat()

	// Step 3: 流式生成（✅ 使用正确的方法名 GetAnswerStream）
	streamReader, err := chatI.GetAnswerStream(ctx, req.ConvID, retriever.Document, req.Question)
	if err != nil {
		g.Log().Error(ctx, "Stream generation failed:", err)
		return nil, err
	}

	// Step 4: 转换为 HTTP SSE 流
	ctx = gctx.New() // 重置上下文，避免请求取消导致流中断
	stream := make(chan string, 10)

	// 后台协程：从 streamReader 读取并转发到 HTTP 流
	go func() {
		defer close(stream)
		defer streamReader.Close()

		for {
			select {
			case <-ctx.Done():
				g.Log().Warning(ctx, "Stream context done:", ctx.Err())
				return
			default:
				// 接收流片段
				msg, err := streamReader.Recv()
				if err != nil {
					if err == io.EOF {
						// 流结束
						g.Log().Debug(ctx, "Stream completed")
						return
					}
					g.Log().Error(ctx, "Stream recv error:", err)
					return
				}

				// 发送到 HTTP 流
				if msg != nil && msg.Content != "" {
					stream <- msg.Content
				}
			}
		}
	}()

	res = &v1.ChatStreamRes{
		Stream:     stream,
		References: retriever.Document,
	}

	return res, nil
}
