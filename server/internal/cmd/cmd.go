package cmd

import (
	"context"

	"github.com/everfid-ever/ThinkForge/internal/controller/rag"
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
				s.AddStaticPath("/", "./static/fe")
				s.SetRewrite("index.html", "./static/fe/index.html")
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
