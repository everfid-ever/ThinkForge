package core

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"time"
)

var system = "You are very skilled at using rag for data retrieval." +
	"Your goal is to perform vectorized retrieval after fully understanding the user's question." +
	"Current time {time_now}" +
	"You need to extract and optimize the search query content." +
	"Please rewrite the query according to the following rules: \n " +
	"- Rewrite the keywords that should be searched based on the user's question and context.\n" +
	"- If time is required, the specific date and time information to be queried will be provided based on the current time.\n" +
	"- Keep your search concise; your search should typically contain no more than three keywords, and at most five.\n" +
	"- Rewrite the keywords according to the current search engine query habits, and directly return the optimized search terms without any additional explanation.\n" +
	"- Try to avoid using the keywords listed below, as previous searches using these keywords did not yield the expected results.\n" +
	"- Keywords already used: {used}\n"

// createTemplate 创建并返回一个配置好的聊天模板
func createTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		// 系统消息模板
		schema.SystemMessage("{system}"),
		// 用户消息模板
		schema.UserMessage(
			"The following are user questions: {question}"),
	)
}

// formatMessages 格式化消息并处理错误
func formatMessages(template prompt.ChatTemplate, data map[string]any) ([]*schema.Message, error) {
	messages, err := template.Format(context.Background(), data)
	if err != nil {
		return nil, fmt.Errorf("template formatting failed: %w", err)
	}
	return messages, nil
}

func getMessages(used string, question string) ([]*schema.Message, error) {
	template := createTemplate()
	data := map[string]any{
		"system":   system,
		"time_now": time.Now().Format(time.RFC3339),
		"question": question,
		"used":     used,
	}
	messages, err := formatMessages(template, data)
	if err != nil {
		return nil, err
	}
	return messages, nil
}
