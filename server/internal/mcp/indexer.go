package mcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/everfid-ever/ThinkForge/core"
	"github.com/everfid-ever/ThinkForge/internal/logic/rag"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

// IndexParam 定义“通过文件路径创建索引”的参数结构
type IndexParam struct {
	URI           string `json:"uri" description:"File path" required:"true"` // // 文件路径或网址（支持 pdf/html/md 等）
	KnowledgeName string `json:"knowledge_name" description:"For the knowledge base name, first retrieve the list using getKnowledgeBaseList and then check if a matching knowledge base exists. If not, generate it automatically based on the user's suggested keywords." required:"true"`
}

// GetIndexerByFilePathTool 定义 MCP 工具元信息
// 工具名称为 “Indexer_by_filepath”，用于根据文件路径执行嵌入任务
func GetIndexerByFilePathTool() *protocol.Tool {
	tool, err := protocol.NewTool("Indexer_by_filepath", "通过文件路径进行文本嵌入", IndexParam{})
	if err != nil {
		g.Log().Errorf(gctx.New(), "Failed to create tool: %v", err)
		return nil
	}
	return tool
}

// HandleIndexerByFilePath 工具执行逻辑（处理函数）
// 当客户端调用该工具时，会执行此函数
func HandleIndexerByFilePath(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var reqData IndexParam
	// 校验请求参数并反序列化
	if err := protocol.VerifyAndUnmarshal(req.RawArguments, &reqData); err != nil {
		return nil, err
	}
	// 获取 RAG 服务实例
	svr := rag.GetRagSvr()
	uri := reqData.URI

	// 构造索引请求参数
	indexReq := &core.IndexReq{
		URI:           uri,
		KnowledgeName: reqData.KnowledgeName,
	}

	// 调用 RAG 服务的索引方法
	ids, err := svr.Index(ctx, indexReq)
	if err != nil {
		return nil, err
	}

	// 构造返回消息
	msg := fmt.Sprintf("index file %s successfully, knowledge_name: %s, doc_ids: %v", uri, reqData.KnowledgeName, ids)

	// 返回标准化结果（MCP 协议格式）
	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: msg,
			},
		},
	}, nil
}

// IndexFileParam 定义“通过 Base64 编码文件内容创建索引”的参数结构
type IndexFileParam struct {
	Filename      string `json:"filename" description:"File name" required:"true"`
	Content       string `json:"content" description:"After the file content is base64 encoded, first use a tool to obtain the base64 information." required:"true"` // 可以是文件路径（pdf，html，md等），也可以是网址文件" required:"true"` // 可以是文件路径（pdf，html，md等），也可以是网址
	KnowledgeName string `json:"knowledge_name" description:"For the knowledge base name, please first retrieve the list using getKnowledgeBaseList and then check if there is a matching knowledge base. If not, generate it automatically based on the user's suggested keywords." required:"true"`
}

// GetIndexerByFileBase64ContentTool 定义 MCP 工具元信息
// 工具名称为 “Indexer_by_base64_file_content”，用于通过 base64 文件内容执行嵌入任务
func GetIndexerByFileBase64ContentTool() *protocol.Tool {
	tool, err := protocol.NewTool("Indexer_by_base64_file_content", "After obtaining the base64 information of the file, upload it, and then embed the content into text.", IndexFileParam{})
	if err != nil {
		g.Log().Errorf(gctx.New(), "Failed to create tool: %v", err)
		return nil
	}
	return tool
}

// HandleIndexerByFileBase64Content 工具执行逻辑（处理函数）
// 当客户端上传 base64 文件内容时，执行解码与后续索引操作
func HandleIndexerByFileBase64Content(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var reqData IndexFileParam
	// 校验请求参数并反序列化
	if err := protocol.VerifyAndUnmarshal(req.RawArguments, &reqData); err != nil {
		return nil, err
	}
	// svr := rag.GetRagSvr()
	// 解码 Base64 文件内容
	decoded, err := base64.StdEncoding.DecodeString(reqData.Content)
	if err != nil {
		return nil, err
	}
	fmt.Println(decoded)
	// indexReq := &gorag.IndexReq{
	// 	URI:           uri,
	// 	KnowledgeName: reqData.KnowledgeName,
	// }
	// ids, err := svr.Index(ctx, indexReq)
	// if err != nil {
	// 	return nil, err
	// }
	// msg := fmt.Sprintf("index file %s successfully, knowledge_name: %s, doc_ids: %v", uri, reqData.KnowledgeName, ids)

	// 当前返回解码后的文本内容，供测试验证使用
	return &protocol.CallToolResult{
		Content: []protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: string(decoded),
			},
		},
	}, nil
}
