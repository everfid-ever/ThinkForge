package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/cloudwego/eino/schema"
)

// WebSearchTool 封装互联网搜索能力，实现 agent.Tool 接口
type WebSearchTool struct {
	enabled    bool
	apiKey     string
	endpoint   string
	maxResults int
}

// WebSearchResult Web 搜索结果
type WebSearchResult struct {
	Results []WebSearchItem `json:"results"`
	Source  string          `json:"source"` // "bing", "disabled", "not_configured", "error"
	Query   string          `json:"query"`
}

// WebSearchItem 单条搜索结果
type WebSearchItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// NewWebSearchTool 创建 WebSearchTool 实例
func NewWebSearchTool(enabled bool, apiKey, endpoint string, maxResults int) *WebSearchTool {
	return &WebSearchTool{
		enabled:    enabled,
		apiKey:     apiKey,
		endpoint:   endpoint,
		maxResults: maxResults,
	}
}

// Name 工具名称
func (t *WebSearchTool) Name() string { return "web_search" }

// Description 工具描述（供 LLM 理解如何调用该工具）
func (t *WebSearchTool) Description() string {
	return "Search the internet for the latest information and real-time data. " +
		"Use this tool to retrieve up-to-date information that may not be present in the knowledge base."
}

// Execute 执行 Web 搜索
func (t *WebSearchTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	if !t.enabled {
		return &WebSearchResult{Results: []WebSearchItem{}, Source: "disabled"}, nil
	}

	query, _ := input["query"].(string)

	maxResults := t.maxResults
	if mr, ok := input["max_results"]; ok {
		switch v := mr.(type) {
		case int:
			maxResults = v
		case float64:
			maxResults = int(v)
		}
	}
	if maxResults <= 0 {
		maxResults = 5
	}

	if query == "" {
		return &WebSearchResult{Results: []WebSearchItem{}, Source: "invalid_input", Query: query}, nil
	}

	if t.apiKey == "" || t.endpoint == "" {
		return &WebSearchResult{Results: []WebSearchItem{}, Source: "not_configured", Query: query}, nil
	}

	// 构建请求 URL
	params := url.Values{}
	params.Set("q", query)
	params.Set("count", strconv.Itoa(maxResults))
	reqURL := t.endpoint + "?" + params.Encode()

	httpClient := &http.Client{Timeout: 10 * time.Second}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		// request creation failed; return empty result per spec (no error propagation)
		return &WebSearchResult{Results: []WebSearchItem{}, Source: "error", Query: query}, nil
	}
	// 兼容 Bing Search API 格式
	httpReq.Header.Set("Ocp-Apim-Subscription-Key", t.apiKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		// network or timeout error; return empty result per spec (no error propagation)
		return &WebSearchResult{Results: []WebSearchItem{}, Source: "error", Query: query}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// body read failure; return empty result per spec (no error propagation)
		return &WebSearchResult{Results: []WebSearchItem{}, Source: "error", Query: query}, nil
	}

	// 解析 Bing 格式响应（字段路径：webPages.value[].name/url/snippet）
	var bingResp struct {
		WebPages struct {
			Value []struct {
				Name    string `json:"name"`
				URL     string `json:"url"`
				Snippet string `json:"snippet"`
			} `json:"value"`
		} `json:"webPages"`
	}
	if err := json.Unmarshal(body, &bingResp); err != nil {
		return &WebSearchResult{Results: []WebSearchItem{}, Source: "bing", Query: query}, nil
	}

	items := make([]WebSearchItem, 0, len(bingResp.WebPages.Value))
	for _, v := range bingResp.WebPages.Value {
		items = append(items, WebSearchItem{
			Title:   v.Name,
			URL:     v.URL,
			Snippet: v.Snippet,
		})
	}

	return &WebSearchResult{Results: items, Source: "bing", Query: query}, nil
}

// ToDocuments 将 WebSearchResult 转换为 schema.Document 列表
// Content 使用搜索摘要（snippet），完整内容需通过 URL 进一步抓取
func (r *WebSearchResult) ToDocuments() []*schema.Document {
	docs := make([]*schema.Document, 0, len(r.Results))
	for _, item := range r.Results {
		docs = append(docs, &schema.Document{
			Content: item.Snippet,
			MetaData: map[string]interface{}{
				"url":    item.URL,
				"title":  item.Title,
				"source": "web_search",
			},
		})
	}
	return docs
}
