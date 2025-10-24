package chat

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/gogf/gf/v2/frame/g"
)

const (
	// role 表示系统角色设定：定义 AI 助手的身份和职责说明
	role = "You are a professional AI assistant that can accurately answer user questions based on the reference information provided."
)

//
// ===================== 模板创建与格式化 =====================
//

// createTemplate 创建并返回一个配置好的聊天模板
// 使用 CloudWeGo Eino 框架的 prompt 组件定义模板结构：
// - 支持系统提示（SystemMessage）
// - 支持上下文历史（MessagesPlaceholder）
// - 支持用户输入（UserMessage）
// 模板采用 FString 模式，将上下文内容直接拼接成可供 LLM 使用的 prompt 字符串。
func createTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		// 系统消息部分：定义 AI 的行为规范与参考内容
		schema.SystemMessage("{role}"+
			"Please strictly abide by the following rules:\n"+
			"1. Answers must be based on the references provided and not rely on external knowledge\n"+
			"2. If the reference content has a clear answer, use the reference content directly to answer\n"+
			"3. If the reference is incomplete or vague, reasonable inferences can be made but the information must be explained\n"+
			"4. If the reference content is completely irrelevant or does not exist, inform the user that the question cannot be answered based on the available information\n"+
			"5. Keep your answers professional, concise, and accurate\n"+
			"6. When necessary, you can quote specific data or original text from the reference content\n\n"+
			"Currently available reference content:\n"+
			"{docs}\n\n"+
			""),

		// 插入聊天历史占位符（chat_history）
		// 用于在多轮对话中传递上下文（历史消息）
		schema.MessagesPlaceholder("chat_history", true),

		// 用户消息模板
		schema.UserMessage("Question: {question}"),
	)
}

// formatMessages 负责执行模板格式化
// 将模板与数据映射（map）结合，生成可直接传递给 LLM 的消息序列
// 这里统一捕获并包装错误，便于上层日志与异常处理。
func formatMessages(template prompt.ChatTemplate, data map[string]any) ([]*schema.Message, error) {
	messages, err := template.Format(context.Background(), data)
	if err != nil {
		return nil, fmt.Errorf("格式化模板失败: %w", err)
	}
	return messages, nil
}

//
// ===================== 文档检索与消息封装 =====================
//

// docsMessages 将检索到的文档内容（docs）与用户问题（question）封装成可供 LLM 输入的消息列表。
// 步骤包括：
// 1. 从会话历史中获取上下文（用于多轮问答）
// 2. 记录当前用户问题
// 3. 构建 Prompt 模板（含 role、docs、question、chat_history）
// 4. 格式化为 LLM 可理解的消息结构
func (x *Chat) docsMessages(ctx context.Context, convID string, docs []*schema.Document, question string) (messages []*schema.Message, err error) {
	// Step 1: 获取指定会话 ID 的历史消息
	chatHistory, err := x.eh.GetHistory(convID, 100)
	if err != nil {
		return
	}

	// Step 2: 将当前用户问题写入历史记录中
	err = x.eh.SaveMessage(&schema.Message{
		Role:    schema.User,
		Content: question,
	}, convID)
	if err != nil {
		return
	}

	// Step 3: 创建聊天模板
	template := createTemplate()

	// Step 4: 打印每个检索到的文档内容，用于调试
	for i, doc := range docs {
		g.Log().Debugf(context.Background(), "docs[%d]: %s", i, doc.Content)
	}

	// Step 5: 组装模板变量
	data := map[string]any{
		"role":         role,        // AI 助手角色设定
		"question":     question,    // 当前用户问题
		"docs":         docs,        // 检索到的知识文档
		"chat_history": chatHistory, // 上下文历史
	}

	// Step 6: 执行模板格式化，将数据填充到模板中
	messages, err = formatMessages(template, data)
	if err != nil {
		return
	}

	// Step 7: 返回构造好的消息序列
	return
}
