package agent

import (
	"context"
	"fmt"
	"strings"
)

// Tool 通用 Tool 接口，ReAct Agent 调用的所有工具都实现此接口
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)
}

// ToolRegistry 工具注册表，支持按名称注册和获取工具
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry 创建空的 ToolRegistry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]Tool)}
}

// Register 注册一个工具
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get 按名称获取工具，不存在则返回 nil, false
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// GetAll 获取所有已注册工具
func (r *ToolRegistry) GetAll() []Tool {
	all := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		all = append(all, t)
	}
	return all
}

// BuildToolDescriptions 构建工具描述字符串，格式化为 LLM 可理解的文本
func (r *ToolRegistry) BuildToolDescriptions() string {
	var sb strings.Builder
	for _, t := range r.tools {
		fmt.Fprintf(&sb, "Tool: %s\nDescription: %s\nInput: {\"query\": \"search keywords\", \"knowledge_name\": \"kb_name\", \"top_k\": 5, \"score\": 0.3}\n\n",
			t.Name(), t.Description())
	}
	return strings.TrimSpace(sb.String())
}
