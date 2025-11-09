package mcp

import (
	"context"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/everfid-ever/ThinkForge/internal/logic/rag"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

// KnowledgeBaseParam 定义工具的输入参数。
// 当前工具无需输入参数，因此为空结构体。
type KnowledgeBaseParam struct {
}

// GetKnowledgeBaseTool 定义一个 MCP 工具 —— “getKnowledgeBaseList”
// 用于从 RAG 系统中获取现有的知识库列表。
// 返回 *protocol.Tool 对象供 MCP Server 注册使用。
func GetKnowledgeBaseTool() *protocol.Tool {
	tool, err := protocol.NewTool("getKnowledgeBaseList", "Get the knowledge base list", KnowledgeBaseParam{})
	if err != nil {
		g.Log().Errorf(gctx.New(), "Failed to create tool: %v", err)
		return nil
	}
	return tool
}

// HandleKnowledgeBase 是工具的执行函数（处理器）
// 当客户端调用 “getKnowledgeBaseList” 工具时，会触发此函数。
func HandleKnowledgeBase(ctx context.Context, toolReq *protocol.CallToolRequest) (res *protocol.CallToolResult, err error) {
	// 获取 RAG 服务实例
	svr := rag.GetRagSvr()

	// 从 RAG 服务中拉取当前所有知识库列表
	list, err := svr.GetKnowledgeBaseList(ctx)

	// 构造响应文本，展示知识库数量与名称列表
	msg := fmt.Sprintf("get %d knowledgeBase", len(list))
	for i, name := range list {
		msg += fmt.Sprintf("\n%d. %s", i+1, name)
	}

	// 将结果封装成 MCP 协议定义的返回类型
	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text", // 响应类型为文本
				Text: msg,    // 响应内容（知识库列表）
			},
		},
	}, nil
}
