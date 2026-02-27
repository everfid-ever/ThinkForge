package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// DecomposeResult 子问题分解结果
type DecomposeResult struct {
	SubQuestions []string // 子问题列表
	Source       string   // "intent"（来自意图分类）/ "llm"（LLM 分解）/ "original"（降级）
}

// SubQuestionDecomposer 子问题分解器
type SubQuestionDecomposer struct {
	model model.BaseChatModel
}

// NewSubQuestionDecomposer 创建子问题分解器
func NewSubQuestionDecomposer(m model.BaseChatModel) *SubQuestionDecomposer {
	return &SubQuestionDecomposer{model: m}
}

// Decompose 将复杂问题分解为子问题列表
// 若 intent.SubQuestions 非空则直接返回；否则调用 LLM 分解；失败则降级为原始问题
func (d *SubQuestionDecomposer) Decompose(ctx context.Context, question string, intent *RAGIntent) ([]string, error) {
	// 若意图分类时已填充子问题，直接使用
	if len(intent.SubQuestions) > 0 {
		return deduplicateAndFilter(intent.SubQuestions), nil
	}

	// 计算最大子问题数
	maxSubQs := normalizeMaxSubQs(intent.EstimatedSteps)

	// 构建分解 Prompt
	systemPrompt := fmt.Sprintf(`You are an expert at breaking down complex questions into simpler sub-questions.

Given a complex question, decompose it into %d or fewer specific, 
searchable sub-questions. Each sub-question should be independently answerable 
through document retrieval.

Output format (JSON array only, no other text):
["sub-question 1", "sub-question 2", ...]

Question type: %s
Complexity: %s`, maxSubQs, intent.Type, intent.Complexity)

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(question),
	}

	resp, err := d.model.Generate(ctx, messages)
	if err != nil {
		// LLM 调用失败，降级为原始问题
		return []string{question}, nil
	}

	// 解析 JSON 数组
	subQuestions, parseErr := parseSubQuestionsJSON(resp.Content)
	if parseErr != nil || len(subQuestions) == 0 {
		// 解析失败，降级为原始问题
		return []string{question}, nil
	}

	return deduplicateAndFilter(subQuestions), nil
}

// DecomposeWithSource 分解子问题并返回来源标识
func (d *SubQuestionDecomposer) DecomposeWithSource(ctx context.Context, question string, intent *RAGIntent) (*DecomposeResult, error) {
	// 若意图分类时已填充子问题
	if len(intent.SubQuestions) > 0 {
		filtered := deduplicateAndFilter(intent.SubQuestions)
		return &DecomposeResult{SubQuestions: filtered, Source: "intent"}, nil
	}

	// 调用 LLM 分解
	maxSubQs := normalizeMaxSubQs(intent.EstimatedSteps)

	systemPrompt := fmt.Sprintf(`You are an expert at breaking down complex questions into simpler sub-questions.

Given a complex question, decompose it into %d or fewer specific, 
searchable sub-questions. Each sub-question should be independently answerable 
through document retrieval.

Output format (JSON array only, no other text):
["sub-question 1", "sub-question 2", ...]

Question type: %s
Complexity: %s`, maxSubQs, intent.Type, intent.Complexity)

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(question),
	}

	resp, err := d.model.Generate(ctx, messages)
	if err != nil {
		return &DecomposeResult{SubQuestions: []string{question}, Source: "original"}, nil
	}

	subQuestions, parseErr := parseSubQuestionsJSON(resp.Content)
	if parseErr != nil || len(subQuestions) == 0 {
		return &DecomposeResult{SubQuestions: []string{question}, Source: "original"}, nil
	}

	filtered := deduplicateAndFilter(subQuestions)
	return &DecomposeResult{SubQuestions: filtered, Source: "llm"}, nil
}

// normalizeMaxSubQs normalizes the max sub-questions count: clamps to [1, 5].
func normalizeMaxSubQs(estimatedSteps int) int {
	if estimatedSteps <= 0 || estimatedSteps > 5 {
		return 5
	}
	return estimatedSteps
}

// parseSubQuestionsJSON 从 LLM 响应中解析 JSON 数组
func parseSubQuestionsJSON(content string) ([]string, error) {
	content = strings.TrimSpace(content)

	// 尝试提取 JSON 数组（LLM 可能会在数组前后有额外文本）
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start >= 0 && end > start {
		content = content[start : end+1]
	}

	var subQuestions []string
	if err := json.Unmarshal([]byte(content), &subQuestions); err != nil {
		return nil, fmt.Errorf("decomposer: failed to parse JSON: %w", err)
	}
	return subQuestions, nil
}

// deduplicateAndFilter 去重并过滤空字符串
func deduplicateAndFilter(questions []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(questions))
	for _, q := range questions {
		q = strings.TrimSpace(q)
		if q == "" || seen[q] {
			continue
		}
		seen[q] = true
		result = append(result, q)
	}
	return result
}
