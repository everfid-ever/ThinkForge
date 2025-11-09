package cmd

import (
	"context"
	"net/http"

	"github.com/ThinkInAIXYZ/go-mcp/server"
	"github.com/ThinkInAIXYZ/go-mcp/transport"
	"github.com/everfid-ever/ThinkForge/internal/controller/rag"
	"github.com/everfid-ever/ThinkForge/internal/mcp"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
)

// Main 命令定义了项目的主入口命令。
// 使用 GoFrame 框架的 gcmd 包，可以通过命令行执行该命令来启动 HTTP 服务
var (
	Main = gcmd.Command{
		Name:  "main",              // 命令名称（执行时使用，如 "main"）
		Usage: "main",              // 命令用法提示
		Brief: "start http server", // 命令简介，描述功能：启动 HTTP 服务器

		// Func 是命令执行的核心逻辑。
		// 当运行 `gf run main` 或 `go run main.go main` 时，将执行此函数。
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {

			// 创建一个默认的 HTTP 服务器实例
			s := g.Server()

			// 定义路由分组（此处为根路径 "/"）
			s.Group("/", func(group *ghttp.RouterGroup) {
				s.AddStaticPath("", "./static/fe/")
				s.SetIndexFiles([]string{"index.html"})
				// 注册全局中间件：自动包装响应格式
				// MiddlewareHandlerResponse 会将返回值统一包装为标准 JSON 响应结构。
				group.Middleware(ghttp.MiddlewareHandlerResponse)

				// 绑定控制器（Controller）
				// rag.NewV1() 返回一个实现了 rag.IRagV1 接口的控制器实例，
				// 用于处理 /v1/chat 和 /v1/retriever 等接口请求。
				group.Bind(
					rag.NewV1(),
				)
			})

			// 启动 HTTP 服务（默认监听 127.0.0.1:8199，或在 config.yaml 中配置）
			s.Run()
			return nil
		},
	}
)

func init() {
	// 向 Main 命令添加一个子命令 "mcp"
	Main.AddCommand(&gcmd.Command{
		Name:  "mcp",              // 子命令名称
		Usage: "mcp",              // 子命令用法提示
		Brief: "start mcp server", // 子命令简介，描述功能：启动 MCP 服务器

		// Func 是子命令执行的核心逻辑。
		// 当运行 `gf run main mcp` 或 `go run main.go main mcp` 时，将执行此函数。
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			// 创建一个基于 Stream HTTP 的 MCP 传输层与对应的 HTTP Handler
			// transport 提供数据流式传输功能，用于服务端和客户端之间进行实时交互
			trans, handler, err := transport.NewStreamableHTTPServerTransportAndHandler()
			if err != nil {
				// 如果创建传输层或处理器失败，记录错误日志并终止程序
				g.Log().Panicf(ctx, "new sse transport and hander with error: %v", err)
			}
			// 使用传输层实例创建一个新的 MCP Server
			mcpServer, _ := server.NewServer(trans)

			// 注册 MCP 工具和对应的处理函数
			mcpServer.RegisterTool(mcp.GetIndexerByFileBase64ContentTool(), mcp.HandleIndexerByFileBase64Content)

			// 注册“根据文件 Base64 内容创建索引”的工具
			// mcpServer.RegisterTool(mcp.GetIndexerByFilePathTool(), mcp.HandleIndexerByFilePath)

			// 注册知识检索工具和对应的处理函数
			mcpServer.RegisterTool(mcp.GetRetrieverTool(), mcp.HandleRetriever)

			// 注册知识库管理工具和对应的处理函数
			mcpServer.RegisterTool(mcp.GetKnowledgeBaseTool(), mcp.HandleKnowledgeBase)
			// 异步启动 MCP 服务器
			go func() {
				mcpServer.Run()
			}()
			defer mcpServer.Shutdown(context.Background())

			// 将 HTTP 路由 “/mcp” 绑定到 handler 的处理函数
			http.Handle("/mcp", handler.HandleMCP())
			return http.ListenAndServe(":8089", nil)
		},
	})
}
