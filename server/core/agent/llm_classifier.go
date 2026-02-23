package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
)

// LLMClassifier 基于 LLM 的意图分类器
type LLMClassifier struct {
	llm model.BaseChatModel
}

// NewLLMClassifier 创建 LLM 分类器
func NewLLMClassifier(llm model.BaseChatModel) *LLMClassifier {
	return &LLMClassifier{
		llm: llm,
	}
}

// Classify 使用 LLM 分类意图
func (lc *LLMClassifier) Classify(ctx context.Context, text string) (*RAGIntent, error) {
	prompt := buildLLMIntentPrompt()

	messages := []*schema.Message{
		schema.SystemMessage(prompt),
		schema.UserMessage(fmt.Sprintf("Question: %s", text)),
	}

	resp, err := lc.llm.Generate(ctx, messages)
	if err != nil {
		g.Log().Errorf(ctx, "LLM classification failed: %v", err)
		return nil, err
	}

	// 解析 JSON 响应
	var intent RAGIntent
	if err := json.Unmarshal([]byte(resp.Content), &intent); err != nil {
		// 尝试提取 JSON（可能有额外文本）
		content := extractJSON(resp.Content)
		if err := json.Unmarshal([]byte(content), &intent); err != nil {
			g.Log().Errorf(ctx, "Failed to parse LLM response: %v", err)
			// 返回默认值
			return &RAGIntent{
				Type:                 RAGIntentUnknown,
				Confidence:           0.5,
				RawText:              text,
				Strategy:             "simple_rag",
				NeedTools:            []string{"rag"},
				EstimatedSteps:       1,
				Complexity:           ComplexityMedium,
				ClassificationMethod: "llm",
				Timestamp:            time.Now().Format(time.RFC3339),
			}, nil
		}
	}

	// 填充元数据
	intent.RawText = text
	intent.ClassificationMethod = "llm"
	intent.Timestamp = time.Now().Format(time.RFC3339)

	return &intent, nil
}

// ClassifyWithContext 带上下文的分类
func (lc *LLMClassifier) ClassifyWithContext(ctx context.Context, text string, history []string) (*RAGIntent, error) {
	prompt := buildLLMIntentPromptWithContext(history)

	messages := []*schema.Message{
		schema.SystemMessage(prompt),
		schema.UserMessage(fmt.Sprintf("Question: %s", text)),
	}

	resp, err := lc.llm.Generate(ctx, messages)
	if err != nil {
		return nil, err
	}

	var intent RAGIntent
	content := extractJSON(resp.Content)
	if err := json.Unmarshal([]byte(content), &intent); err != nil {
		return lc.Classify(ctx, text) // 降级到无上下文版本
	}

	intent.RawText = text
	intent.ClassificationMethod = "llm_context"
	intent.Timestamp = time.Now().Format(time.RFC3339)

	return &intent, nil
}

// ClassifyBatch 批量分类
func (lc *LLMClassifier) ClassifyBatch(ctx context.Context, texts []string) ([]*RAGIntent, error) {
	results := make([]*RAGIntent, len(texts))
	for i, text := range texts {
		intent, err := lc.Classify(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = intent
	}
	return results, nil
}

func buildLLMIntentPrompt() string {
	return `You are an expert intent classifier for a RAG (Retrieval-Augmented Generation) system.

Your task is to analyze user questions and classify them into specific intent types.

Intent Types:
1. simple_qa: Simple factual questions (e.g., "What is RAG?")
2. fact_check: Verification questions (e.g., "Is it true that...")
3. multi_hop_qa: Multi-step reasoning (e.g., "Why does X cause Y?")
4. causal_reasoning: Cause-effect analysis (e.g., "What caused...")
5. procedural: How-to questions (e.g., "How to configure...")
6. comparison: Comparison analysis (e.g., "Compare A vs B")
7. summarization: Summary requests (e.g., "Summarize...")
8. aggregation: Data aggregation (e.g., "How many...", "Calculate...")
9. trend_analysis: Trend analysis (e.g., "What's the trend...")
10. hybrid_search: Needs external data (e.g., "Latest news about...")
11. realtime_query: Real-time data (e.g., "Current status...")
12. code_generation: Code generation (e.g., "Write code to...")
13. content_creation: Content creation (e.g., "Write an article about...")
14. clarification: Unclear questions
15. unknown: Cannot classify

Output JSON format:
{
  "type": "intent_type",
  "confidence": 0.85,
  "strategy": "simple_rag|react_agent|hybrid",
  "need_tools": ["rag", "web_search"],
  "estimated_steps": 3,
  "complexity": "simple|medium|complex",
  "requires_external": false,
  "knowledge_domains": ["machine_learning"],
  "sub_questions": ["sub question 1", "sub question 2"]
}

Examples:

Question: "什么是RAG?"
{
  "type": "simple_qa",
  "confidence": 0.95,
  "strategy": "simple_rag",
  "need_tools": ["rag"],
  "estimated_steps": 1,
  "complexity": "simple",
  "requires_external": false,
  "knowledge_domains": ["nlp"],
  "sub_questions": []
}

Question: "对比 Elasticsearch 和 Milvus 的性能，并给出推荐"
{
  "type": "comparison",
  "confidence": 0.9,
  "strategy": "react_agent",
  "need_tools": ["rag", "web_search"],
  "estimated_steps": 4,
  "complexity": "complex",
  "requires_external": false,
  "knowledge_domains": ["database", "vector_search"],
  "sub_questions": [
    "Elasticsearch 的性能特点",
    "Milvus 的性能特点",
    "两者性能对比",
    "推荐方案"
  ]
}

Question: "最新的 GPT-5 有什么新功能?"
{
  "type": "hybrid_search",
  "confidence": 0.88,
  "strategy": "hybrid",
  "need_tools": ["rag", "web_search"],
  "estimated_steps": 2,
  "complexity": "medium",
  "requires_external": true,
  "knowledge_domains": ["ai", "llm"],
  "sub_questions": []
}

Now analyze the following question and return ONLY the JSON output, no additional text:`
}

func buildLLMIntentPromptWithContext(history []string) string {
	basePrompt := buildLLMIntentPrompt()

	if len(history) > 0 {
		contextText := "\n\nConversation History:\n"
		for i, msg := range history {
			contextText += fmt.Sprintf("%d. %s\n", i+1, msg)
		}
		contextText += "\nConsider the context when classifying the current question."

		return basePrompt + contextText
	}

	return basePrompt
}

func extractJSON(content string) string {
	// 尝试提取 JSON 块
	start := -1
	end := -1

	for i, c := range content {
		if c == '{' && start == -1 {
			start = i
		}
		if c == '}' {
			end = i + 1
		}
	}

	if start != -1 && end != -1 && end > start {
		return content[start:end]
	}

	return content
}
