package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/everfid-ever/ThinkForge/core"
)

// RagTool 将 core.Rag.Retrieve 封装为 ReAct Agent 可调用的工具
type RagTool struct {
	ragSvr        *core.Rag
	knowledgeName string
	topK          int
	score         float64
}

// RagToolInput ReAct Agent 调用 RAG 工具的输入参数
type RagToolInput struct {
	Query         string  `json:"query"`          // 检索关键词
	KnowledgeName string  `json:"knowledge_name"` // 知识库名称（可覆盖默认值）
	TopK          int     `json:"top_k"`          // 返回文档数量
	Score         float64 `json:"score"`          // 相关性阈值
}

// RagToolOutput RAG 工具的执行结果
type RagToolOutput struct {
	Documents []*schema.Document `json:"documents"`
	Count     int                `json:"count"`
}

// NewRagTool 创建 RAG 工具实例
func NewRagTool(ragSvr *core.Rag, knowledgeName string, topK int, score float64) *RagTool {
	return &RagTool{
		ragSvr:        ragSvr,
		knowledgeName: knowledgeName,
		topK:          topK,
		score:         score,
	}
}

// Name 工具名称
func (t *RagTool) Name() string { return "rag_retriever" }

// Description 工具描述（供 LLM 理解如何调用该工具）
func (t *RagTool) Description() string {
	return "Search documents from a knowledge base using semantic similarity. " +
		"Use this tool to retrieve relevant information for answering questions."
}

// Execute 执行 RAG 检索
func (t *RagTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// 将 map 序列化再反序列化为 RagToolInput，以便统一处理
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("rag_tool: failed to marshal input: %w", err)
	}
	var toolInput RagToolInput
	if err = json.Unmarshal(data, &toolInput); err != nil {
		return nil, fmt.Errorf("rag_tool: failed to unmarshal input: %w", err)
	}

	// 应用默认值
	if toolInput.Query == "" {
		return nil, fmt.Errorf("rag_tool: query is required")
	}
	if toolInput.KnowledgeName == "" {
		toolInput.KnowledgeName = t.knowledgeName
	}
	if toolInput.TopK <= 0 {
		toolInput.TopK = t.topK
	}
	if toolInput.Score <= 0 {
		toolInput.Score = t.score
	}

	docs, err := t.ragSvr.Retrieve(ctx, &core.RetrieveReq{
		Query:         toolInput.Query,
		TopK:          toolInput.TopK,
		Score:         toolInput.Score,
		KnowledgeName: toolInput.KnowledgeName,
	})
	if err != nil {
		return nil, fmt.Errorf("rag_tool: retrieve failed: %w", err)
	}

	return &RagToolOutput{
		Documents: docs,
		Count:     len(docs),
	}, nil
}
