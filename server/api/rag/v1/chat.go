package v1

import (
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
)

// ChatReq 定义了聊天请求的数据结构。
// 该结构用于 /v1/chat 接口的 POST 请求体。
type ChatReq struct {
	g.Meta `path:"/v1/chat" method:"post" tags:"rag"`
	// g.Meta 是 GoFrame 框架用于绑定接口元数据的结构标签。
	// path: 指定请求路径
	// method: 指定请求方法
	// tags: 用于接口文档分组标签（例如 swagger 中显示为 “rag” 分类）

	ConvID   string  `json:"conv_id"`  // 会话ID，用于标识同一对话的上下文。
	Question string  `json:"question"` // 用户提出的问题内容。
	TopK     int     `json:"top_k"`    // 检索文档的数量（Top K 条）；当需要基于文档检索时传入。
	Score    float64 `json:"score"`    // 文档检索的相关性分数阈值；当需要检索文档时传入。
}

// ChatRes 定义了聊天接口的响应结构。
// 用于返回生成的答案结果。
type ChatRes struct {
	g.Meta     `mime:"application/json"` // 指定返回的数据格式为 JSON。
	Answer     string                    `json:"answer"` // 生成的答案文本内容。
	References []*schema.Document        `json:"references"`
}

// ChatStreamReq 流式输出请求
type ChatStreamReq struct {
	g.Meta        `path:"/v1/chat/stream" method:"post" tags:"rag"`
	ConvID        string  `json:"conv_id" v:"required"` // 会话id
	Question      string  `json:"question" v:"required"`
	KnowledgeName string  `json:"knowledge_name" v:"required"`
	TopK          int     `json:"top_k"` // 默认为5
	Score         float64 `json:"score"` // 默认为0.2
}

// ChatStreamRes 流式输出响应
type ChatStreamRes struct {
	g.Meta `mime:"text/event-stream"`
	// 流式输出不需要返回具体内容，内容通过HTTP响应流返回
}
