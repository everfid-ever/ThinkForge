package rag

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"
	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/core/agent"
	"github.com/gogf/gf/v2/frame/g"
)

// AgenticChat Agentic RAG 聊天入口
func (c *ControllerV1) AgenticChat(ctx context.Context, req *v1.AgenticChatReq) (res *v1.AgenticChatRes, err error) {
	startTime := time.Now()
	g.Log().Infof(ctx, "🚀 Agentic RAG started: %s", req.Question)

	// Step 1: 获取分类器
	var classifier agent.IntentClassifier
	if req.UseRuleOnly {
		classifier = agent.NewHybridIntentClassifierRuleOnly()
		g.Log().Info(ctx, "Using rule-only classifier")
	} else {
		// 使用全局单例分类器
		classifier = agent.GetClassifier()
	}

	// Step 2: 意图识别
	intent, err := classifier.Classify(ctx, req.Question)
	if err != nil {
		g.Log().Errorf(ctx, "Intent classification failed: %v", err)
		return nil, err
	}

	g.Log().Infof(ctx, "🎯 Intent: type=%s, confidence=%.2f, strategy=%s, complexity=%s, method=%s",
		intent.Type, intent.Confidence, intent.Strategy, intent.Complexity, intent.ClassificationMethod)

	// Step 3: 根据意图选择策略
	var answer string
	var references []*schema.Document
	var reasoningSteps []agent.ReasoningStep

	switch intent.Strategy {
	case "simple_rag":
		answer, references, err = c.simpleRAGStrategy(ctx, req, intent)

	case "react_agent":
		answer, references, reasoningSteps, err = c.reactAgentStrategy(ctx, req, intent)

	case "hybrid":
		answer, references, err = c.hybridStrategy(ctx, req, intent)

	default:
		answer, references, err = c.simpleRAGStrategy(ctx, req, intent)
	}

	if err != nil {
		g.Log().Errorf(ctx, "Strategy execution failed: %v", err)
		return nil, err
	}

	// Step 4: 构造响应
	executionTime := time.Since(startTime).Milliseconds()

	res = &v1.AgenticChatRes{
		Answer:         answer,
		References:     references,
		Intent:         intent,
		ReasoningSteps: reasoningSteps,
		ExecutionTime:  executionTime,
	}

	g.Log().Infof(ctx, "✅ Agentic RAG completed in %dms", executionTime)

	return res, nil
}

// IntentClassify 意图分类接口
func (c *ControllerV1) IntentClassify(ctx context.Context, req *v1.IntentClassifyReq) (res *v1.IntentClassifyRes, err error) {
	classifier := agent.GetClassifier()

	var intent *agent.RAGIntent
	if len(req.History) > 0 {
		intent, err = classifier.ClassifyWithContext(ctx, req.Question, req.History)
	} else {
		intent, err = classifier.Classify(ctx, req.Question)
	}

	if err != nil {
		return nil, err
	}

	res = &v1.IntentClassifyRes{
		Intent: intent,
	}

	return res, nil
}

// simpleRAGStrategy 简单 RAG 策略
func (c *ControllerV1) simpleRAGStrategy(ctx context.Context, req *v1.AgenticChatReq, intent *agent.RAGIntent) (string, []*schema.Document, error) {
	chatReq := &v1.ChatReq{
		ConvID:   req.ConvID,
		Question: req.Question,
		TopK:     5,
		Score:    0.2,
	}

	chatRes, err := c.Chat(ctx, chatReq)
	if err != nil {
		return "", nil, err
	}

	return chatRes.Answer, chatRes.References, nil
}

// reactAgentStrategy ReAct Agent 策略
func (c *ControllerV1) reactAgentStrategy(ctx context.Context, req *v1.AgenticChatReq, intent *agent.RAGIntent) (string, []*schema.Document, []agent.ReasoningStep, error) {
	answer, references, err := c.simpleRAGStrategy(ctx, req, intent)
	if err != nil {
		return "", nil, nil, err
	}

	// TODO: 实现完整 ReAct 推理循环
	steps := []agent.ReasoningStep{
		{
			Step:      1,
			Type:      "thought",
			Content:   fmt.Sprintf("Question requires %s analysis with %d estimated steps", intent.Complexity, intent.EstimatedSteps),
			Timestamp: time.Now().Format(time.RFC3339),
		},
		{
			Step:      2,
			Type:      "action",
			Content:   "Searching knowledge base: " + req.KnowledgeName,
			Timestamp: time.Now().Format(time.RFC3339),
		},
		{
			Step:      3,
			Type:      "observation",
			Content:   fmt.Sprintf("Found %d relevant documents", len(references)),
			Timestamp: time.Now().Format(time.RFC3339),
		},
		{
			Step:      4,
			Type:      "thought",
			Content:   "Synthesizing answer based on retrieved documents",
			Timestamp: time.Now().Format(time.RFC3339),
		},
	}

	return answer, references, steps, nil
}

// hybridStrategy 混合策略
func (c *ControllerV1) hybridStrategy(ctx context.Context, req *v1.AgenticChatReq, intent *agent.RAGIntent) (string, []*schema.Document, error) {
	// TODO: 实现 RAG + Web 搜索混合策略
	g.Log().Info(ctx, "Hybrid strategy: combining RAG with external search")
	return c.simpleRAGStrategy(ctx, req, intent)
}
