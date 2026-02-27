package rag

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"
	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/core/agent"
	"github.com/everfid-ever/ThinkForge/internal/logic/chat"
	"github.com/gogf/gf/v2/frame/g"
)

// Chat 智能 RAG 统一入口（支持传统模式和 Agentic 模式）
func (c *ControllerV1) Chat(ctx context.Context, req *v1.ChatReq) (res *v1.ChatRes, err error) {
	startTime := time.Now()
	g.Log().Infof(ctx, "🚀 Smart RAG: %s", req.Question)

	useAgentic := req.EnableAgentic || req.KnowledgeName != ""

	// 🔍 重要：调试日志，便于排查
	g.Log().Infof(ctx, "📊 Agentic mode: %v (EnableAgentic=%v, KnowledgeName=%q)",
		useAgentic, req.EnableAgentic, req.KnowledgeName)

	if !useAgentic {
		g.Log().Info(ctx, "Using legacy RAG mode (no KnowledgeName)")
		return c.legacyRAG(ctx, req)
	}

	// ===== Agentic RAG 模式 =====

	// Step 1: 意图识别
	classifier := c.getClassifier(req)
	intent, err := classifier.Classify(ctx, req.Question)
	if err != nil {
		g.Log().Warningf(ctx, "Intent classification failed: %v, fallback to legacy", err)
		return c.legacyRAG(ctx, req)
	}

	g.Log().Infof(ctx, "🎯 Intent: type=%s, confidence=%.2f, strategy=%s",
		intent.Type, intent.Confidence, intent.Strategy)

	// Step 2: 置信度极低 → 无法判断意图，走快速通道兜底
	// 注意：不应将 ComplexitySimple 作为 fast-path 的条件，
	// 简单问题会通过 intent.Strategy == "simple_rag" 在 Step 3 中正确路由。
	if intent.Confidence < 0.3 {
		g.Log().Infof(ctx, "Very low confidence (%.2f), using fast-path (simple RAG)", intent.Confidence)
		answer, references, err := c.executeSimpleRAG(ctx, req)
		if err != nil {
			return nil, err
		}
		return c.buildChatResponse(answer, references, intent, time.Since(startTime), req), nil
	}

	// Step 3: 根据策略执行
	var answer string
	var references []*schema.Document
	var reasoningSteps []agent.ReasoningStep

	switch intent.Strategy {
	case "simple_rag":
		answer, references, err = c.executeSimpleRAG(ctx, req)

	case "react_agent":
		answer, references, reasoningSteps, err = c.executeReActAgent(ctx, req, intent)

	case "hybrid":
		answer, references, err = c.executeHybridSearch(ctx, req, intent)

	default:
		answer, references, err = c.executeSimpleRAG(ctx, req)
	}

	if err != nil {
		g.Log().Errorf(ctx, "Strategy execution failed: %v, fallback to legacy", err)
		return c.legacyRAG(ctx, req)
	}

	// Step 4: 构造响应
	executionTime := time.Since(startTime)
	res = c.buildChatResponse(answer, references, intent, executionTime, req)

	// 可选：返回推理步骤
	if req.ReturnSteps && len(reasoningSteps) > 0 {
		res.ReasoningSteps = reasoningSteps
	}

	g.Log().Infof(ctx, "✅ Completed in %dms using %s", executionTime.Milliseconds(), intent.Strategy)

	return res, nil
}

// ===== 策略执行方法 =====

// executeSimpleRAG 执行简单 RAG 策略
func (c *ControllerV1) executeSimpleRAG(ctx context.Context, req *v1.ChatReq) (string, []*schema.Document, error) {
	// Step 1: 检索
	retriever, err := c.Retriever(ctx, &v1.RetrieverReq{
		Question:      req.Question,
		TopK:          req.TopK,
		Score:         req.Score,
		KnowledgeName: req.KnowledgeName,
	})
	if err != nil {
		return "", nil, err
	}

	// Step 2: 生成
	chatI := chat.GetChat()
	answer, err := chatI.GetAnswer(ctx, req.ConvID, retriever.Document, req.Question)
	if err != nil {
		return "", nil, err
	}

	return answer, retriever.Document, nil
}

// executeReActAgent 执行 ReAct Agent 策略
func (c *ControllerV1) executeReActAgent(ctx context.Context, req *v1.ChatReq, intent *agent.RAGIntent) (string, []*schema.Document, []agent.ReasoningStep, error) {
	g.Log().Infof(ctx, "Executing ReAct agent (estimated steps: %d)", intent.EstimatedSteps)

	// 当前先调用 simple RAG，未来扩展为完整 ReAct 循环
	answer, references, err := c.executeSimpleRAG(ctx, req)
	if err != nil {
		return "", nil, nil, err
	}

	// 生成推理步骤（展示 Agent 思考过程）
	steps := c.generateReasoningSteps(intent, len(references), req.KnowledgeName)

	return answer, references, steps, nil
}

// executeHybridSearch 执行混合检索策略（RAG + 外部数据）
func (c *ControllerV1) executeHybridSearch(ctx context.Context, req *v1.ChatReq, intent *agent.RAGIntent) (string, []*schema.Document, error) {
	g.Log().Info(ctx, "Executing hybrid search (RAG + external)")

	// TODO: 实现混合检索
	// 当前降级到 simple RAG
	return c.executeSimpleRAG(ctx, req)
}

// legacyRAG 传统 RAG 实现（完全保留原逻辑）
func (c *ControllerV1) legacyRAG(ctx context.Context, req *v1.ChatReq) (res *v1.ChatRes, err error) {
	retriever, err := c.Retriever(ctx, &v1.RetrieverReq{
		Question:      req.Question,
		TopK:          req.TopK,
		Score:         req.Score,
		KnowledgeName: req.KnowledgeName,
	})
	if err != nil {
		return nil, err
	}

	chatI := chat.GetChat()
	answer, err := chatI.GetAnswer(ctx, req.ConvID, retriever.Document, req.Question)
	if err != nil {
		return nil, err
	}

	res = &v1.ChatRes{
		Answer:        answer,
		References:    retriever.Document,
		Strategy:      "legacy_rag",
		ExecutionTime: 0,
	}

	return res, nil
}

// ===== 辅助方法 =====

// getClassifier 获取分类器
func (c *ControllerV1) getClassifier(req *v1.ChatReq) agent.IntentClassifier {
	if req.UseRuleOnly {
		return agent.NewHybridIntentClassifierRuleOnly()
	}
	return agent.GetClassifier()
}

// buildChatResponse 构造响应
func (c *ControllerV1) buildChatResponse(
	answer string,
	references []*schema.Document,
	intent *agent.RAGIntent,
	executionTime time.Duration,
	req *v1.ChatReq,
) *v1.ChatRes {
	res := &v1.ChatRes{
		Answer:        answer,
		References:    references,
		Strategy:      intent.Strategy,
		ExecutionTime: executionTime.Milliseconds(),
	}

	// 可选：返回意图信息
	if req.ReturnIntent {
		res.Intent = intent
	}

	return res
}

// generateReasoningSteps 生成推理步骤
func (c *ControllerV1) generateReasoningSteps(intent *agent.RAGIntent, docCount int, knowledgeName string) []agent.ReasoningStep {
	now := time.Now().Format(time.RFC3339)
	return []agent.ReasoningStep{
		{
			Step:      1,
			Type:      "thought",
			Content:   fmt.Sprintf("Question type: %s, complexity: %s", intent.Type, intent.Complexity),
			Timestamp: now,
		},
		{
			Step:    2,
			Type:    "action",
			Content: "Searching knowledge base: " + knowledgeName,
			ActionInput: map[string]interface{}{
				"tool":      "rag_retriever",
				"kb_name":   knowledgeName,
				"estimated": intent.EstimatedSteps,
			},
			Timestamp: now,
		},
		{
			Step:      3,
			Type:      "observation",
			Content:   fmt.Sprintf("Found %d relevant documents", docCount),
			Timestamp: now,
		},
		{
			Step:      4,
			Type:      "thought",
			Content:   "Synthesizing answer based on retrieved context",
			Timestamp: now,
		},
	}
}
