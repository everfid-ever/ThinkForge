package mcp

import (
	"context"
	"fmt"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

// RetrieverParam 定义“检索知识库文档”的输入参数结构。
// 该结构由 MCP 协议自动解析并传入 HandleRetriever。
type RetrieverParam struct {
	Question      string  `json:"question" description:"Questions asked by users" required:"true"`
	KnowledgeName string  `json:"knowledge_name" description:"For the knowledge base name, please first retrieve the list using getKnowledgeBaseList and then check if there is a knowledge base that matches the user's suggested keywords." required:"true"`
	TopK          int     `json:"top_k" description:"The default number of search results is 5." required:"false"`               // 默认为5
	Score         float64 `json:"score"  description:"The score threshold for search results defaults to 0.2." required:"false"` // 默认为0.2
}

// GetRetrieverTool 定义一个 MCP 工具 “retriever”
// 用于通过语义检索，从指定知识库中获取相关文档内容。
func GetRetrieverTool() *protocol.Tool {
	tool, err := protocol.NewTool("retriever", "Retrieve Knowledge Base Documents", RetrieverParam{})
	if err != nil {
		g.Log().Errorf(gctx.New(), "Failed to create tool: %v", err)
		return nil
	}
	return tool
}

// HandleRetriever 是工具的处理函数。
// 当客户端调用 “retriever” 工具时，会执行此函数。
func HandleRetriever(ctx context.Context, toolReq *protocol.CallToolRequest) (res *protocol.CallToolResult, err error) {
	var req RetrieverParam

	// 解析并验证来自客户端的原始参数
	if err := protocol.VerifyAndUnmarshal(toolReq.RawArguments, &req); err != nil {
		return nil, err
	}
	retriever, err := c.Retriever(ctx, &v1.RetrieverReq{
		Question:      req.Question,
		TopK:          req.TopK,
		Score:         req.Score,
		KnowledgeName: req.KnowledgeName,
	})
	if err != nil {
		return nil, err
	}
	// 获取返回的文档列表
	docs := retriever.Document
	// 构造结果文本，展示检索到的文档数量及内容摘要
	msg := fmt.Sprintf("retrieve %d documents", len(docs))
	for i, doc := range docs {
		msg += fmt.Sprintf("\n%d. score: %.2f, content: %s", i+1, doc.Score(), doc.Content)
	}
	// 封装返回结果（符合 MCP 协议格式）
	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: msg,
			},
		},
	}, nil
}
