package rag

import (
	"context"
	"github.com/everfid-ever/ThinkForge/internal/logic/chat"

	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/core/agent"
	"github.com/gogf/gf/v2/frame/g"
)

// Chat 智能 RAG 接口（自动选择策略）
func (c *ControllerV1) Chat(ctx context.Context, req *v1.ChatReq) (res *v1.ChatRes, err error) {
	// Step 1: 智能意图识别（轻量级，仅规则）
	classifier := agent.NewHybridIntentClassifierRuleOnly()
	intent, err := classifier.Classify(ctx, req.Question)
	if err != nil {
		g.Log().Warningf(ctx, "Intent classification failed, fallback to legacy RAG: %v", err)
		return c.legacyChat(ctx, req)
	}

	g.Log().Debugf(ctx, "Intent: %s, Confidence: %.2f", intent.Type, intent.Confidence)

	// Step 2: 简单问题 → 快速通道（传统 RAG）
	if intent.Complexity == agent.ComplexitySimple && intent.Confidence > 0.7 {
		g.Log().Info(ctx, "Using fast-path (legacy RAG)")
		return c.legacyChat(ctx, req)
	}

	// Step 3: 复杂问题 → Agentic 通道
	g.Log().Info(ctx, "Using intelligent path (Agentic RAG)")

	agenticReq := &v1.AgenticChatReq{
		ConvID:        req.ConvID,
		Question:      req.Question,
		KnowledgeName: "",   // 需要从 req 中获取或配置
		UseRuleOnly:   true, // 保持快速
		MaxIterations: 3,
	}

	agenticRes, err := c.AgenticChat(ctx, agenticReq)
	if err != nil {
		// Agentic 失败，降级到传统 RAG
		g.Log().Warningf(ctx, "Agentic RAG failed, fallback: %v", err)
		return c.legacyChat(ctx, req)
	}

	// 转换为 ChatRes 格式
	res = &v1.ChatRes{
		Answer:     agenticRes.Answer,
		References: agenticRes.References,
	}

	return res, nil
}

// legacyChat 传统 RAG 实现（保留原逻辑）
func (c *ControllerV1) legacyChat(ctx context.Context, req *v1.ChatReq) (res *v1.ChatRes, err error) {
	// 原 Chat 方法的实现
	retriever, err := c.Retriever(ctx, &v1.RetrieverReq{
		Question: req.Question,
		TopK:     req.TopK,
		Score:    req.Score,
	})
	if err != nil {
		return
	}

	chatI := chat.GetChat()
	answer, err := chatI.GetAnswer(ctx, req.ConvID, retriever.Document, req.Question)
	if err != nil {
		return
	}

	res = &v1.ChatRes{
		Answer:     answer,
		References: retriever.Document,
	}
	return
}
