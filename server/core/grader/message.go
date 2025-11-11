package grader

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

// createRetrieverTemplate 判断检索到的文档是否足够回答用户问题
func createRetrieverTemplate() prompt.ChatTemplate {
	// 创建模板，使用 FString 格式
	return prompt.FromMessages(schema.FString,
		// 系统消息模板
		schema.SystemMessage(
			"You are an expert who assesses whether the retrieved documents are sufficient to answer user questions."+
				"Please carefully understand the user's question first."+
				"If the retrieved documents are sufficient to answer the user's question, please select 'yes'."+
				"If the retrieved documents are insufficient to answer the user's question, please select 'no'."+
				"Do not provide any further explanation.",
		),
		// 用户消息模板
		schema.UserMessage(
			"These are the retrieved documents.: \n"+
				"{document} \n\n"+
				"This is a user's problem: {question}"),
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

func retrieverMessages(docs []*schema.Document, question string) ([]*schema.Message, error) {
	document := ""
	for i, doc := range docs {
		document += fmt.Sprintf("docs[%d]: %s", i, doc.Content)
	}
	template := createRetrieverTemplate()
	data := map[string]any{
		"question": question,
		"document": document,
	}
	messages, err := formatMessages(template, data)
	if err != nil {
		return nil, err
	}
	return messages, nil
}
