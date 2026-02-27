package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// MultiHopConfig 多跳推理配置
type MultiHopConfig struct {
	Model         model.BaseChatModel // LLM 实例
	Registry      *ToolRegistry       // 工具注册表
	MaxSubQs      int                 // 最大子问题数，默认 5
	MergeStrategy string              // 合并策略："sequential"（顺序）/ "parallel"（并行，预留）
}

// SubQuestionResult 单个子问题的执行结果
type SubQuestionResult struct {
	SubQuestion string
	Documents   []*schema.Document
	Answer      string
	Step        int
}

// MultiHopResult 多跳推理最终结果
type MultiHopResult struct {
	FinalAnswer    string
	AllReferences  []*schema.Document  // 所有子问题检索到的文档（去重）
	SubResults     []SubQuestionResult // 每个子问题的结果
	ReasoningSteps []ReasoningStep     // 完整推理步骤
}

// MultiHopExecutor 多跳推理执行器
type MultiHopExecutor struct {
	config *MultiHopConfig
}

// NewMultiHopExecutor 创建多跳推理执行器
func NewMultiHopExecutor(config *MultiHopConfig) *MultiHopExecutor {
	if config.MaxSubQs <= 0 {
		config.MaxSubQs = 5
	}
	if config.MergeStrategy == "" {
		config.MergeStrategy = "sequential"
	}
	return &MultiHopExecutor{config: config}
}

// Run 执行多跳推理
func (e *MultiHopExecutor) Run(ctx context.Context, intent *RAGIntent, originalQuestion string, knowledgeName string, topK int, score float64) (*MultiHopResult, error) {
	var steps []ReasoningStep
	stepNum := 0

	// Step 1: 分解子问题
	decomposer := NewSubQuestionDecomposer(e.config.Model)
	decompResult, err := decomposer.DecomposeWithSource(ctx, originalQuestion, intent)
	if err != nil {
		return nil, fmt.Errorf("multi_hop: decompose failed: %w", err)
	}

	subQuestions := decompResult.SubQuestions
	// 限制最大子问题数
	if len(subQuestions) > e.config.MaxSubQs {
		subQuestions = subQuestions[:e.config.MaxSubQs]
	}

	// 记录分解思考步骤
	stepNum++
	now := time.Now().Format(time.RFC3339)
	steps = append(steps, ReasoningStep{
		Step:      stepNum,
		Type:      "thought",
		Content:   fmt.Sprintf("Decomposing question into %d sub-questions (source: %s)", len(subQuestions), decompResult.Source),
		Timestamp: now,
	})

	// 获取 rag_retriever 工具
	ragTool, hasTool := e.config.Registry.Get("rag_retriever")

	// Step 2: 逐个执行子问题（sequential）
	var subResults []SubQuestionResult
	for i, subQ := range subQuestions {
		now = time.Now().Format(time.RFC3339)

		// thought 步骤
		stepNum++
		steps = append(steps, ReasoningStep{
			Step:      stepNum,
			Type:      "thought",
			Content:   fmt.Sprintf("Analyzing sub-question %d/%d: %q", i+1, len(subQuestions), subQ),
			Timestamp: now,
		})

		// action 步骤
		actionInput := map[string]interface{}{
			"query":          subQ,
			"knowledge_name": knowledgeName,
			"top_k":          topK,
			"score":          score,
		}
		stepNum++
		steps = append(steps, ReasoningStep{
			Step:        stepNum,
			Type:        "action",
			Content:     "rag_retriever",
			ActionInput: actionInput,
			Timestamp:   now,
		})

		// 执行检索
		var subDocs []*schema.Document
		if !hasTool {
			// 工具不存在，跳过
			now = time.Now().Format(time.RFC3339)
			stepNum++
			steps = append(steps, ReasoningStep{
				Step:      stepNum,
				Type:      "observation",
				Content:   fmt.Sprintf("Found 0 documents for sub-question %d (tool not available)", i+1),
				Timestamp: now,
			})
			continue
		}

		toolResult, execErr := ragTool.Execute(ctx, actionInput)
		now = time.Now().Format(time.RFC3339)
		if execErr != nil {
			// 单个子问题检索失败，跳过继续
			stepNum++
			steps = append(steps, ReasoningStep{
				Step:      stepNum,
				Type:      "observation",
				Content:   fmt.Sprintf("Found 0 documents for sub-question %d (error: %v)", i+1, execErr),
				Timestamp: now,
			})
			continue
		}

		// 提取文档（使用 JSON 序列化/反序列化，避免 import cycle）
		subDocs = extractDocsFromToolResult(toolResult)

		// observation 步骤
		stepNum++
		steps = append(steps, ReasoningStep{
			Step:      stepNum,
			Type:      "observation",
			Content:   fmt.Sprintf("Found %d documents for sub-question %d", len(subDocs), i+1),
			Timestamp: now,
		})

		subResults = append(subResults, SubQuestionResult{
			SubQuestion: subQ,
			Documents:   subDocs,
			Step:        i + 1,
		})
	}

	// 若所有子问题均失败（无任何结果），返回 error
	if len(subResults) == 0 {
		return nil, fmt.Errorf("multi_hop: all sub-questions failed to retrieve documents")
	}

	// Step 3: 合并所有文档（去重，保留最高 score）
	allDocs := mergeAndDeduplicateDocs(subResults, topK*2)

	// Step 4: 构建合成 Prompt，生成最终答案
	finalAnswer, synthErr := e.synthesizeAnswer(ctx, originalQuestion, subResults)
	if synthErr != nil {
		// LLM 合成失败，拼接所有子问题答案（若有）或返回 error
		var parts []string
		for _, sr := range subResults {
			if sr.Answer != "" {
				parts = append(parts, sr.Answer)
			}
		}
		if len(parts) > 0 {
			finalAnswer = strings.Join(parts, "\n\n")
		} else {
			return nil, fmt.Errorf("multi_hop: synthesize failed: %w", synthErr)
		}
	}

	// 记录最终答案步骤
	now = time.Now().Format(time.RFC3339)
	stepNum++
	steps = append(steps, ReasoningStep{
		Step:      stepNum,
		Type:      "final_answer",
		Content:   fmt.Sprintf("Synthesized answer from %d sub-questions", len(subResults)),
		Timestamp: now,
	})

	return &MultiHopResult{
		FinalAnswer:    finalAnswer,
		AllReferences:  allDocs,
		SubResults:     subResults,
		ReasoningSteps: steps,
	}, nil
}

// synthesizeAnswer 调用 LLM 合成最终答案
func (e *MultiHopExecutor) synthesizeAnswer(ctx context.Context, originalQuestion string, subResults []SubQuestionResult) (string, error) {
	// 构建子问题与文档摘要
	var sb strings.Builder
	for i, sr := range subResults {
		fmt.Fprintf(&sb, "Sub-question %d: %s\n", i+1, sr.SubQuestion)
		sb.WriteString("Documents: ")
		for j, doc := range sr.Documents {
			if j > 0 {
				sb.WriteString("\n")
			}
			content := doc.Content
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			sb.WriteString(content)
		}
		sb.WriteString("\n\n")
	}

	systemPrompt := fmt.Sprintf(`You are a professional AI assistant synthesizing answers from multiple retrieved documents.

Original question: %s

Sub-questions and retrieved context:
%s

Instructions:
1. Synthesize a comprehensive answer to the original question using all the retrieved information
2. If sub-questions have contradictory information, note the discrepancy
3. Be concise but complete
4. Cite specific information from the documents when relevant`, originalQuestion, sb.String())

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(originalQuestion),
	}

	resp, err := e.config.Model.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("multi_hop: llm synthesize failed: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}

// mergeAndDeduplicateDocs 合并所有子问题文档，按 doc.ID 去重，保留最高 score，按 score 降序排列
func mergeAndDeduplicateDocs(subResults []SubQuestionResult, maxDocs int) []*schema.Document {
	// 按 ID 去重，保留最高 score
	best := make(map[string]*schema.Document)
	for _, sr := range subResults {
		for _, doc := range sr.Documents {
			if existing, ok := best[doc.ID]; !ok {
				best[doc.ID] = doc
			} else if doc.Score() > existing.Score() {
				best[doc.ID] = doc
			}
		}
	}

	// 收集并排序
	merged := make([]*schema.Document, 0, len(best))
	for _, doc := range best {
		merged = append(merged, doc)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score() > merged[j].Score()
	})

	// 截取前 maxDocs 个
	if maxDocs > 0 && len(merged) > maxDocs {
		merged = merged[:maxDocs]
	}
	return merged
}

// extractDocsFromToolResult 从工具返回值中提取文档列表（通过 JSON 序列化，避免 import cycle）
func extractDocsFromToolResult(result interface{}) []*schema.Document {
	data, err := json.Marshal(result)
	if err != nil {
		return nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	docsRaw, ok := raw["documents"]
	if !ok {
		return nil
	}
	docsData, err := json.Marshal(docsRaw)
	if err != nil {
		return nil
	}
	var docs []*schema.Document
	if err := json.Unmarshal(docsData, &docs); err != nil {
		return nil
	}
	return docs
}
