package v1

import (
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
)

// RetrieverReq 定义了文档检索接口（/v1/retriever）的请求结构。
// 该接口通常用于从知识库中根据问题检索相关文档。
type RetrieverReq struct {
	g.Meta `path:"/v1/retriever" method:"post" tags:"rag"`
	// g.Meta 是 GoFrame 框架用于接口元信息绑定的标签字段。
	// path: 指定接口路径（此处为 /v1/retriever）
	// method: 指定请求方法为 POST
	// tags: 用于接口文档的分组标签（如 Swagger 中显示为 "rag" 分组）

	Question      string  `json:"question" v:"required"`       // 用户输入的问题内容（必填）
	TopK          int     `json:"top_k"`                       // 需要返回的文档数量（默认为 5）
	Score         float64 `json:"score"`                       // 文档相关性评分阈值（默认为 0.2）
	KnowledgeName string  `json:"knowledge_name" v:"required"` // 目标知识库名称（必填）
}

// RetrieverRes 定义了文档检索接口的响应结构。
// 用于返回与问题最相关的文档内容。
type RetrieverRes struct {
	g.Meta   `mime:"application/json"` // 指定返回的数据格式为 JSON
	Document []*schema.Document        `json:"document"` // 检索得到的文档列表（来自 eino/schema 的文档结构）
}
