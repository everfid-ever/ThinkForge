package agent

import (
	"context"
	"github.com/cloudwego/eino/schema"
)

// RAGIntentType 针对 RAG 场景的专用意图类型
type RAGIntentType string

const (
	// 基础意图
	RAGIntentSimpleQA  RAGIntentType = "simple_qa"  // 简单问答：单次检索即可回答
	RAGIntentFactCheck RAGIntentType = "fact_check" // 事实核查：需要验证信息准确性

	// 复杂推理意图
	RAGIntentMultiHopQA      RAGIntentType = "multi_hop_qa"     // 多跳推理：需要多次检索才能回答
	RAGIntentCausalReasoning RAGIntentType = "causal_reasoning" // 因果推理：分析原因和结果
	RAGIntentProcedural      RAGIntentType = "procedural"       // 过程性问题：如何做某事

	// 分析类意图
	RAGIntentComparison    RAGIntentType = "comparison"     // 对比分析：比较多个对象
	RAGIntentSummarization RAGIntentType = "summarization"  // 摘要生成：总结大量信息
	RAGIntentAggregation   RAGIntentType = "aggregation"    // 数据聚合：统计、计算
	RAGIntentTrendAnalysis RAGIntentType = "trend_analysis" // 趋势分析：识别模式和趋势

	// 混合检索意图
	RAGIntentHybridSearch  RAGIntentType = "hybrid_search"  // 混合检索：需要 RAG + 外部数据
	RAGIntentRealtimeQuery RAGIntentType = "realtime_query" // 实时查询：需要最新数据

	// 创作类意图
	RAGIntentCodeGeneration  RAGIntentType = "code_generation"  // 代码生成
	RAGIntentContentCreation RAGIntentType = "content_creation" // 内容创作（文章、方案等）

	// 特殊意图
	RAGIntentClarification RAGIntentType = "clarification" // 问题澄清：问题不明确
	RAGIntentUnknown       RAGIntentType = "unknown"       // 未知意图
)

// ComplexityLevel 问题复杂度级别
type ComplexityLevel string

const (
	ComplexitySimple  ComplexityLevel = "simple"  // 简单：1-2步可解决
	ComplexityMedium  ComplexityLevel = "medium"  // 中等：3-5步
	ComplexityComplex ComplexityLevel = "complex" // 复杂：5+步或需要多工具协同
)

// RAGIntent RAG 专用意图结构
type RAGIntent struct {
	// 基础信息
	Type       RAGIntentType `json:"type"`       // 意图类型
	Confidence float64       `json:"confidence"` // 置信度 (0-1)
	RawText    string        `json:"raw_text"`   // 原始问题

	// 策略信息
	Strategy       string          `json:"strategy"`        // 推荐策略：simple_rag, react_agent, hybrid
	NeedTools      []string        `json:"need_tools"`      // 需要的工具：["rag", "web_search", "calculator"]
	EstimatedSteps int             `json:"estimated_steps"` // 预估推理步数
	Complexity     ComplexityLevel `json:"complexity"`      // 复杂度

	// 上下文信息
	RequiresExternal bool     `json:"requires_external"` // 是否需要外部数据
	KnowledgeDomains []string `json:"knowledge_domains"` // 涉及的知识领域
	SubQuestions     []string `json:"sub_questions"`     // 分解的子问题

	// 约束条件
	TimeConstraint  *TimeConstraint  `json:"time_constraint,omitempty"`  // 时间约束
	ScopeConstraint *ScopeConstraint `json:"scope_constraint,omitempty"` // 范围约束

	// 元数据
	ClassificationMethod string `json:"classification_method"` // 分类方法：rule/llm/hybrid
	Timestamp            string `json:"timestamp"`             // 分类时间
}

// TimeConstraint 时间约束
type TimeConstraint struct {
	StartTime string `json:"start_time,omitempty"` // 开始时间
	EndTime   string `json:"end_time,omitempty"`   // 结束时间
	Relative  string `json:"relative,omitempty"`   // 相对时间：last_week, today, etc.
}

// ScopeConstraint 范围约束
type ScopeConstraint struct {
	KnowledgeBases []string `json:"knowledge_bases,omitempty"` // 限定知识库
	Categories     []string `json:"categories,omitempty"`      // 限定分类
	Entities       []string `json:"entities,omitempty"`        // 限定实体
}

// IntentClassifier 意图分类器接口
type IntentClassifier interface {
	// Classify 分类单条消息
	Classify(ctx context.Context, text string) (*RAGIntent, error)

	// ClassifyWithContext 带上下文的分类
	ClassifyWithContext(ctx context.Context, text string, history []string) (*RAGIntent, error)

	// ClassifyBatch 批量分类
	ClassifyBatch(ctx context.Context, texts []string) ([]*RAGIntent, error)
}

// IntentRouter 意图路由器接口
type IntentRouter interface {
	// Route 根据意图选择处理策略
	Route(ctx context.Context, intent *RAGIntent) (Strategy, error)
}

// Strategy 处理策略接口
type Strategy interface {
	// Execute 执行策略
	Execute(ctx context.Context, intent *RAGIntent, question string) (*StrategyResult, error)

	// Name 策略名称
	Name() string
}

// StrategyResult 策略执行结果
type StrategyResult struct {
	Answer         string             `json:"answer"`
	References     []*schema.Document `json:"references"`
	ReasoningSteps []ReasoningStep    `json:"reasoning_steps,omitempty"`
	Confidence     float64            `json:"confidence"`
	TokensUsed     int                `json:"tokens_used"`
	ExecutionTime  int64              `json:"execution_time_ms"`
}

// ReasoningStep 推理步骤
type ReasoningStep struct {
	Step        int                    `json:"step"`
	Type        string                 `json:"type"` // thought/action/observation
	Content     string                 `json:"content"`
	ActionInput map[string]interface{} `json:"action_input,omitempty"`
	Timestamp   string                 `json:"timestamp"`
}
