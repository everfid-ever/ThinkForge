package v1

import (
	"github.com/cloudwego/eino/schema"
	"github.com/everfid-ever/ThinkForge/core/agent"
	"github.com/gogf/gf/v2/frame/g"
)

// AgenticChatReq Agentic RAG 请求（v1 - Advanced）
//
// 适用场景：
// - 复杂推理问题（如："为什么X会导致Y?"）
// - 对比分析（如："对比A和B的优缺点"）
// - 需要最新数据（如："最新的GPT-5有什么功能?"）
//
// 优势：
// - ✅ 自动意图识别
// - ✅ 多策略智能路由
// - ✅ ReAct 推理能力
// - ✅ 完整推理链可追溯
type AgenticChatReq struct {
	g.Meta        `path:"/v1/agentic-chat" method:"post" tags:"rag"`
	ConvID        string   `json:"conv_id"`
	Question      string   `json:"question" v:"required"`
	KnowledgeName string   `json:"knowledge_name" v:"required"`
	EnabledTools  []string `json:"enabled_tools"` // ["rag", "web_search", "calculator"]
	MaxIterations int      `json:"max_iterations" d:"5"`
	UseRuleOnly   bool     `json:"use_rule_only" d:"false"` // 仅规则分类（更快）
}

// AgenticChatRes Agentic RAG 响应
type AgenticChatRes struct {
	g.Meta         `mime:"application/json"`
	Answer         string                `json:"answer"`            // 答案
	References     []*schema.Document    `json:"references"`        // 引用文档
	Intent         *agent.RAGIntent      `json:"intent"`            // 识别的意图
	ReasoningSteps []agent.ReasoningStep `json:"reasoning_steps"`   // 推理步骤
	TokensUsed     int                   `json:"tokens_used"`       // Token 使用量
	ExecutionTime  int64                 `json:"execution_time_ms"` // 执行时间（毫秒）
}

// IntentClassifyReq 意图分类请求
type IntentClassifyReq struct {
	g.Meta   `path:"/v1/intent/classify" method:"post" tags:"rag"`
	Question string   `json:"question" v:"required"` // 问题
	History  []string `json:"history"`               // 历史对话
}

// IntentClassifyRes 意图分类响应
type IntentClassifyRes struct {
	g.Meta `mime:"application/json"`
	Intent *agent.RAGIntent `json:"intent"` // 意图
}
