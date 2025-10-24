package chat

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai" // Eino 扩展：OpenAI 模型封装
	"github.com/cloudwego/eino/components/model"            // Eino 通用模型接口定义
	"github.com/cloudwego/eino/schema"                      // Eino 消息与文档数据结构定义
	"github.com/gogf/gf/v2/frame/g"                         // GoFrame 日志与配置模块
	"github.com/gogf/gf/v2/os/gctx"                         // GoFrame Context 工具
	"github.com/wangle201210/chat-history/eino"             // 外部包：Eino 聊天历史管理器
)

//
// ===================== Chat 全局单例定义 =====================
//

// chat 为包级单例，全局共享 Chat 对象
var chat *Chat

// Chat 结构体封装了一个完整的对话生成引擎，包含：
// - cm：底层 Chat 模型实例（例如 OpenAI GPT）
// - eh：聊天历史记录管理器（支持多轮上下文）
type Chat struct {
	cm model.BaseChatModel // 底层大语言模型（LLM）
	eh *eino.History       // 聊天历史管理器
}

// GetChat 返回全局 Chat 实例，供外部模块调用（例如 ControllerV1）
func GetChat() *Chat {
	return chat
}

//
// ===================== 初始化流程 =====================
//

// init() 在包加载时自动执行：
// 1. 从配置文件读取 OpenAI API 参数
// 2. 创建 ChatModel 实例
// 3. 初始化聊天历史管理器
// 4. 注入全局 chat 单例
func init() {
	ctx := gctx.New()

	// 从配置文件中读取 LLM 参数
	c, err := newChat(&openai.ChatModelConfig{
		APIKey:  g.Cfg().MustGet(ctx, "chat.apiKey").String(),  // API 密钥
		BaseURL: g.Cfg().MustGet(ctx, "chat.baseURL").String(), // API 地址
		Model:   g.Cfg().MustGet(ctx, "chat.model").String(),   // 模型名称（例如 gpt-4）
	})
	if err != nil {
		g.Log().Fatalf(ctx, "newChat failed, err=%v", err)
		return
	}

	// 初始化历史记录管理器（存储路径来自配置文件）
	c.eh = eino.NewEinoHistory(
		g.Cfg().MustGet(ctx, "chat.history").String(),
	)

	// 注册为全局单例
	chat = c
}

//
// ===================== Chat 实例创建 =====================
//

// newChat 根据配置创建一个新的 Chat 实例
// 封装 Eino 的 openai.NewChatModel，用于建立与 LLM 的通信通道。
func newChat(cfg *openai.ChatModelConfig) (res *Chat, err error) {
	chatModel, err := openai.NewChatModel(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	return &Chat{cm: chatModel}, nil
}

//
// ===================== 核心生成逻辑 =====================
//

// GetAnswer 是 Chat 模块的核心方法：根据问题与检索结果生成回答。
// 主要步骤：
// 1. 生成对话上下文（包括历史记录 + 文档）
// 2. 调用 LLM 生成回答
// 3. 保存回答到聊天历史
// 4. 返回最终回答文本
func (x *Chat) GetAnswer(ctx context.Context, convID string, docs []*schema.Document, question string) (answer string, err error) {
	// Step 1: 构造 LLM 所需的输入消息（模板 + 历史 + 文档）
	messages, err := x.docsMessages(ctx, convID, docs, question)
	if err != nil {
		return "", err
	}

	// Step 2: 调用底层 LLM 生成答案
	result, err := generate(ctx, x.cm, messages)
	if err != nil {
		return "", fmt.Errorf("生成答案失败: %w", err)
	}

	// Step 3: 将 LLM 输出保存到对话历史中
	err = x.eh.SaveMessage(result, convID)
	if err != nil {
		g.Log().Error(ctx, "save assistant message err: %v", err)
		return
	}

	// Step 4: 返回最终内容
	return result.Content, nil
}

//
// ===================== LLM 调用封装 =====================
//

// generate 封装 LLM 的生成调用逻辑
// 输入为消息数组（包含系统提示、历史对话、文档上下文等）
// 输出为 LLM 生成的单条消息（assistant 角色）
func generate(ctx context.Context, llm model.BaseChatModel, in []*schema.Message) (message *schema.Message, err error) {
	message, err = llm.Generate(ctx, in)
	if err != nil {
		err = fmt.Errorf("llm generate failed: %v", err)
		return
	}
	return
}
