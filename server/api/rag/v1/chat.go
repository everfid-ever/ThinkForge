package v1

import (
	"github.com/cloudwego/eino/schema"
	"github.com/everfid-ever/ThinkForge/core/agent"
	"github.com/gogf/gf/v2/frame/g"
)

// ===== Chat 普通对话 =====

// ChatReq 智能 RAG 请求（统一接口）
type ChatReq struct {
	g.Meta `path:"/v1/chat" method:"post" tags:"rag"`

	// ===== 基础参数 =====
	ConvID        string `json:"conv_id"`                     // 会话 ID
	Question      string `json:"question" v:"required"`       // 用户问题
	KnowledgeName string `json:"knowledge_name" v:"required"` // 知识库名称

	// ===== 检索参数 =====
	TopK  int     `json:"top_k" d:"5"`   // 返回文档数量
	Score float64 `json:"score" d:"0.2"` // 相关性阈值

	// ===== Agentic 参数（新增，可选） =====
	EnableAgentic bool     `json:"enable_agentic" d:"true"` // 是否启用智能路由（默认开启）
	UseRuleOnly   bool     `json:"use_rule_only" d:"true"`  // 仅使用规则分类（更快）
	MaxIterations int      `json:"max_iterations" d:"5"`    // ReAct 最大推理轮数
	EnabledTools  []string `json:"enabled_tools,omitempty"` // 启用的工具（空=自动）

	// ===== 调试参数 =====
	ReturnIntent bool `json:"return_intent" d:"false"` // 是否返回意图信息
	ReturnSteps  bool `json:"return_steps" d:"false"`  // 是否返回推理步骤
}

// ChatRes 智能 RAG 响应
type ChatRes struct {
	g.Meta `mime:"application/json"`

	// ===== 核心返回 =====
	Answer     string             `json:"answer"`     // 答案
	References []*schema.Document `json:"references"` // 引用文档

	// ===== 元信息 =====
	Strategy      string `json:"strategy"`          // 使用的策略
	ExecutionTime int64  `json:"execution_time_ms"` // 执行时间（毫秒）

	// ===== 可选返回（调试用） =====
	Intent         *agent.RAGIntent      `json:"intent,omitempty"`          // 意图分析
	ReasoningSteps []agent.ReasoningStep `json:"reasoning_steps,omitempty"` // 推理步骤
}

// ===== ChatStream 流式对话 =====

// ChatStreamReq 流式对话请求
type ChatStreamReq struct {
	g.Meta `path:"/v1/chat/stream" method:"post" tags:"rag"`

	// ===== 基础参数 =====
	ConvID        string `json:"conv_id"`
	Question      string `json:"question" v:"required"`
	KnowledgeName string `json:"knowledge_name" v:"required"`

	// ===== 检索参数 =====
	TopK  int     `json:"top_k" d:"5"`
	Score float64 `json:"score" d:"0.2"`

	// ===== Agentic 参数 =====
	EnableAgentic bool `json:"enable_agentic" d:"false"` // 流式默认关闭智能模式（性能考虑）
}

// ChatStreamRes 流式对话响应
type ChatStreamRes struct {
	g.Meta `mime:"text/event-stream"`

	Stream     <-chan string      `json:"-"`          // 流式响应通道
	References []*schema.Document `json:"references"` // 引用文档
}
