package rag

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/core/agent"
	"github.com/everfid-ever/ThinkForge/core/agent/tools"
	"github.com/everfid-ever/ThinkForge/internal/logic/chat"
	ragLogic "github.com/everfid-ever/ThinkForge/internal/logic/rag"
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
	g.Log().Infof(ctx, "🤖 Executing ReAct agent (intent=%s, estimated_steps=%d)", intent.Type, intent.EstimatedSteps)

	// 获取 LLM 实例
	chatModel := agent.GetChatModel()
	if chatModel == nil {
		g.Log().Warning(ctx, "ChatModel not available for ReAct, fallback to simple RAG")
		answer, references, err := c.executeSimpleRAG(ctx, req)
		if err != nil {
			return "", nil, nil, err
		}
		return answer, references, nil, nil
	}

	// 获取 RAG 服务
	ragSvr := ragLogic.GetRagSvr()

	// 构建工具注册表
	registry := agent.NewToolRegistry()
	ragTool := tools.NewRagTool(ragSvr, req.KnowledgeName, req.TopK, req.Score)
	registry.Register(ragTool)

	// 构建 ReAct 执行器
	maxIter := req.MaxIterations
	if maxIter <= 0 {
		maxIter = 5
	}
	executor := agent.NewReactExecutor(&agent.ReactConfig{
		MaxIterations:  maxIter,
		Model:          chatModel,
		Registry:       registry,
		EnableMultiHop: agent.IsMultiHopIntent(intent),
	})

	// 执行 ReAct 循环
	result, err := executor.Run(ctx, intent, req.Question, req.KnowledgeName, req.TopK, req.Score)
	if err != nil {
		g.Log().Errorf(ctx, "ReAct execution failed: %v, fallback to simple RAG", err)
		answer, references, err2 := c.executeSimpleRAG(ctx, req)
		if err2 != nil {
			return "", nil, nil, err2
		}
		return answer, references, nil, nil
	}

	g.Log().Infof(ctx, "✅ ReAct completed: %d steps, %d references", len(result.ReasoningSteps), len(result.References))
	return result.Answer, result.References, result.ReasoningSteps, nil
}

// executeHybridSearch 执行混合检索策略（RAG + Web Search 并行）
func (c *ControllerV1) executeHybridSearch(ctx context.Context, req *v1.ChatReq, intent *agent.RAGIntent) (string, []*schema.Document, error) {
	g.Log().Infof(ctx, "🔍 Executing hybrid search (intent=%s)", intent.Type)

	// 从配置读取 Web Search 参数
	webEnabledVar, _ := g.Cfg().Get(ctx, "agent.web_search.enabled", false)
	apiKeyVar, _ := g.Cfg().Get(ctx, "agent.web_search.api_key", "")
	endpointVar, _ := g.Cfg().Get(ctx, "agent.web_search.endpoint", "")
	webConfigEnabled := webEnabledVar.Bool()
	apiKey := apiKeyVar.String()
	endpoint := endpointVar.String()

	doWebSearch := c.isWebSearchEnabled(ctx, req, intent) && webConfigEnabled

	// 1. 并行执行 RAG 检索 和 Web Search
	var (
		ragDocs []*schema.Document
		webDocs []*schema.Document
		ragErr  error
		webErr  error
		wg      sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		retriever, err := c.Retriever(ctx, &v1.RetrieverReq{
			Question:      req.Question,
			TopK:          req.TopK,
			Score:         req.Score,
			KnowledgeName: req.KnowledgeName,
		})
		if err != nil {
			ragErr = err
			return
		}
		ragDocs = retriever.Document
	}()

	if doWebSearch {
		wg.Add(1)
		go func() {
			defer wg.Done()
			topK := req.TopK
			if topK <= 0 {
				topK = 5
			}
			webTool := tools.NewWebSearchTool(true, apiKey, endpoint, topK)
			input := map[string]interface{}{
				"query":       req.Question,
				"max_results": topK,
			}
			result, err := webTool.Execute(ctx, input)
			if err != nil {
				webErr = err
				return
			}
			if searchResult, ok := result.(*tools.WebSearchResult); ok {
				webDocs = searchResult.ToDocuments()
			}
		}()
	}

	wg.Wait()

	// 2. 错误处理
	if ragErr != nil && webErr != nil {
		return "", nil, fmt.Errorf("hybrid search: both RAG and web search failed: rag=%w, web=%v", ragErr, webErr)
	}
	if ragErr != nil {
		g.Log().Warningf(ctx, "⚠️ RAG retrieval failed, using only web results: %v", ragErr)
	}
	if webErr != nil {
		g.Log().Warningf(ctx, "⚠️ Web search failed, using only RAG results: %v", webErr)
	}

	g.Log().Infof(ctx, "📚 Hybrid results: RAG=%d, Web=%d", len(ragDocs), len(webDocs))

	// 3. 合并去重排序截断
	mergedDocs := c.deduplicateAndMergeDocs(ragDocs, webDocs, intent, req.TopK)

	// 4. 空结果降级到 simple RAG
	if len(mergedDocs) == 0 {
		g.Log().Info(ctx, "No hybrid results, fallback to simple RAG")
		return c.executeSimpleRAG(ctx, req)
	}

	g.Log().Infof(ctx, "🔀 Merged docs: %d (intent=%s)", len(mergedDocs), intent.Type)

	// 5. 调用 LLM 生成答案
	chatI := chat.GetChat()
	answer, err := chatI.GetAnswer(ctx, req.ConvID, mergedDocs, req.Question)
	if err != nil {
		return "", nil, err
	}

	return answer, mergedDocs, nil
}

// deduplicateAndMergeDocs 合并去重并排序文档列表
// 对于 RAGIntentRealtimeQuery，web 结果优先；其他情况 RAG 结果优先
func (c *ControllerV1) deduplicateAndMergeDocs(ragDocs, webDocs []*schema.Document, intent *agent.RAGIntent, topK int) []*schema.Document {
	var primary, secondary []*schema.Document
	if intent.Type == agent.RAGIntentRealtimeQuery {
		// 实时查询：web 结果排在前面
		primary = webDocs
		secondary = ragDocs
	} else {
		// 其他情况：RAG 结果优先
		primary = ragDocs
		secondary = webDocs
	}

	seen := make(map[string]bool)
	result := make([]*schema.Document, 0, len(primary)+len(secondary))

	for _, doc := range append(primary, secondary...) {
		if doc == nil {
			continue
		}
		// 以内容前 100 字符作为去重 key
		key := doc.Content
		if len(key) > 100 {
			key = key[:100]
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, doc)
		}
	}

	// 截断：合并后最多保留 topK*2 条（不超过 20 条）
	maxDocs := topK * 2
	if maxDocs > 20 {
		maxDocs = 20
	}
	if maxDocs <= 0 {
		maxDocs = 10
	}
	if len(result) > maxDocs {
		result = result[:maxDocs]
	}

	return result
}

// isWebSearchEnabled 判断当前请求是否应启用 Web Search
func (c *ControllerV1) isWebSearchEnabled(_ context.Context, req *v1.ChatReq, intent *agent.RAGIntent) bool {
	// 检查意图是否需要外部数据
	intentNeedsWeb := intent.RequiresExternal ||
		intent.Type == agent.RAGIntentHybridSearch ||
		intent.Type == agent.RAGIntentRealtimeQuery
	if !intentNeedsWeb {
		return false
	}

	// EnabledTools 为空表示允许所有工具；否则需明确包含 "web_search"
	if len(req.EnabledTools) == 0 {
		return true
	}
	for _, tool := range req.EnabledTools {
		if tool == "web_search" {
			return true
		}
	}
	return false
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
