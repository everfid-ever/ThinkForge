package rag

import "github.com/everfid-ever/ThinkForge/api/rag"

// ControllerV1 是 RAG（Retrieval-Augmented Generation）模块的 V1 版本控制器。
// 它负责实现 rag.IRagV1 接口中定义的两个核心功能：
//   - Chat：处理聊天/问答请求
//   - Retriever：处理文档检索请求
//
// 控制器通常作为业务逻辑的入口层（Controller Layer），
// 负责接收 HTTP 请求、调用业务逻辑（Service Layer）、
// 并将结果封装成标准响应返回。
type ControllerV1 struct{}

// NewV1 创建并返回一个 IRagV1 接口的实现实例。
// 在 cmd/main.go 中，会通过 group.Bind(rag.NewV1()) 自动绑定到路由系统。
func NewV1() rag.IRagV1 {
	return &ControllerV1{}
}
