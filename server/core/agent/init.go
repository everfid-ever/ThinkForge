package agent

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

var (
	globalChatModel   model.BaseChatModel
	globalClassifier  IntentClassifier
	globalIntentCache = NewIntentCache(5*time.Minute, 1000)
	once              sync.Once
	classifierOnce    sync.Once
	chatModelInitErr  error
	classifierInitErr error
)

// GetIntentCache 获取全局意图缓存
func GetIntentCache() *IntentCache {
	return globalIntentCache
}

// init 在包加载时自动初始化
func init() {
	ctx := gctx.New()

	// 初始化 ChatModel
	_, err := InitChatModel()
	if err != nil {
		g.Log().Warningf(ctx, "Failed to initialize ChatModel for agent: %v (will use rule-only classifier)", err)
	}
}

// InitChatModel 初始化全局 ChatModel
func InitChatModel() (model.BaseChatModel, error) {
	once.Do(func() {
		ctx := gctx.New()

		// 从配置文件读取
		apiKey := g.Cfg().MustGet(ctx, "chat.apiKey").String()
		baseURL := g.Cfg().MustGet(ctx, "chat.baseURL").String()
		modelName := g.Cfg().MustGet(ctx, "chat.model").String()

		// 如果配置为空，尝试从环境变量读��
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if baseURL == "" {
			baseURL = os.Getenv("OPENAI_BASE_URL")
		}
		if modelName == "" {
			modelName = "gpt-3.5-turbo" // 默认模型
		}

		if apiKey == "" {
			chatModelInitErr = fmt.Errorf("OPENAI_API_KEY not configured")
			g.Log().Warning(ctx, "ChatModel initialization skipped: API key not found")
			return
		}

		config := &openai.ChatModelConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
			Model:   modelName,
		}

		globalChatModel, chatModelInitErr = openai.NewChatModel(context.Background(), config)
		if chatModelInitErr != nil {
			g.Log().Errorf(ctx, "Failed to initialize ChatModel: %v", chatModelInitErr)
		} else {
			g.Log().Infof(ctx, "ChatModel initialized successfully: model=%s", modelName)
		}
	})

	return globalChatModel, chatModelInitErr
}

// GetChatModel 获取全局 ChatModel 实例
func GetChatModel() model.BaseChatModel {
	if globalChatModel == nil {
		globalChatModel, _ = InitChatModel()
	}
	return globalChatModel
}

// GetClassifier 获取全局分类器实例（单例）
func GetClassifier() IntentClassifier {
	classifierOnce.Do(func() {
		chatModel := GetChatModel()

		if chatModel == nil {
			// ChatModel 不可用，使用纯规则分类器
			globalClassifier = NewHybridIntentClassifierRuleOnly()
			g.Log().Info(context.Background(), "Using rule-only intent classifier")
		} else {
			// ChatModel 可用，使用混合分类器
			globalClassifier = NewHybridIntentClassifier(chatModel)
			g.Log().Info(context.Background(), "Using hybrid intent classifier (rule + LLM)")
		}
	})

	return globalClassifier
}

// ReloadClassifier 重新加载分类器（用于动态切换）
func ReloadClassifier(useLLM bool) IntentClassifier {
	if useLLM {
		chatModel := GetChatModel()
		if chatModel != nil {
			globalClassifier = NewHybridIntentClassifier(chatModel)
		} else {
			g.Log().Warning(context.Background(), "ChatModel not available, using rule-only")
			globalClassifier = NewHybridIntentClassifierRuleOnly()
		}
	} else {
		globalClassifier = NewHybridIntentClassifierRuleOnly()
	}

	return globalClassifier
}
