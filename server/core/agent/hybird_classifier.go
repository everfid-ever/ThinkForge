package agent

import (
	"context"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/gogf/gf/v2/frame/g"
)

// HybridIntentClassifier 混合意图分类器：规则优先 + LLM 兜底
type HybridIntentClassifier struct {
	ruleClassifier *RuleBasedClassifier
	llmClassifier  *LLMClassifier
	useLLM         bool

	// 配置参数
	highConfidenceThreshold float64 // 高置信度阈值（跳过 LLM）
	lowConfidenceThreshold  float64 // 低置信度阈值（使用 LLM）
}

// NewHybridIntentClassifier 创建混合分类器
func NewHybridIntentClassifier(llm model.BaseChatModel) *HybridIntentClassifier {
	return &HybridIntentClassifier{
		ruleClassifier:          NewRuleBasedClassifier(),
		llmClassifier:           NewLLMClassifier(llm),
		useLLM:                  true,
		highConfidenceThreshold: 0.7,
		lowConfidenceThreshold:  0.5,
	}
}

// NewHybridIntentClassifierRuleOnly 创建仅使用规则的分类器
func NewHybridIntentClassifierRuleOnly() *HybridIntentClassifier {
	return &HybridIntentClassifier{
		ruleClassifier: NewRuleBasedClassifier(),
		useLLM:         false,
	}
}

// Classify 混合分类：规则优先，LLM 兜底
func (hc *HybridIntentClassifier) Classify(ctx context.Context, text string) (*RAGIntent, error) {
	startTime := time.Now()

	// Step 1: 规则分类
	ruleResult, err := hc.ruleClassifier.Classify(ctx, text)
	if err != nil {
		return nil, err
	}

	g.Log().Debugf(ctx, "Rule classification: type=%s, confidence=%.2f", ruleResult.Type, ruleResult.Confidence)

	// Step 2: 高置信度 → 直接返回规则结果
	if ruleResult.Confidence >= hc.highConfidenceThreshold {
		g.Log().Infof(ctx, "High confidence rule match (%.2f), skip LLM", ruleResult.Confidence)
		ruleResult.ClassificationMethod = "rule"
		return ruleResult, nil
	}

	// Step 3: 低置信度或未知意图 → 使用 LLM
	if hc.useLLM && (ruleResult.Confidence < hc.lowConfidenceThreshold || ruleResult.Type == RAGIntentUnknown) {
		g.Log().Infof(ctx, "Low confidence (%.2f) or unknown, using LLM", ruleResult.Confidence)

		llmResult, err := hc.llmClassifier.Classify(ctx, text)
		if err != nil {
			g.Log().Warningf(ctx, "LLM classification failed: %v, fallback to rule", err)
			return ruleResult, nil
		}

		// Step 4: LLM 结果更好 → 使用 LLM 结果
		if llmResult.Confidence > ruleResult.Confidence {
			g.Log().Infof(ctx, "LLM result better (%.2f > %.2f)", llmResult.Confidence, ruleResult.Confidence)
			llmResult.ClassificationMethod = "hybrid_llm"

			elapsed := time.Since(startTime).Milliseconds()
			g.Log().Debugf(ctx, "Classification time: %dms", elapsed)

			return llmResult, nil
		}
	}

	// Step 5: 中等置信度或 LLM 结果不理想 → 返回规则结果
	ruleResult.ClassificationMethod = "hybrid_rule"

	elapsed := time.Since(startTime).Milliseconds()
	g.Log().Debugf(ctx, "Classification time: %dms", elapsed)

	return ruleResult, nil
}

// ClassifyWithContext 带上下文的分类
func (hc *HybridIntentClassifier) ClassifyWithContext(ctx context.Context, text string, history []string) (*RAGIntent, error) {
	// 规则分类
	ruleResult, err := hc.ruleClassifier.ClassifyWithContext(ctx, text, history)
	if err != nil {
		return nil, err
	}

	// 高置信度 → 直接返回
	if ruleResult.Confidence >= hc.highConfidenceThreshold {
		return ruleResult, nil
	}

	// 低置信度 → LLM
	if hc.useLLM && ruleResult.Confidence < hc.lowConfidenceThreshold {
		llmResult, err := hc.llmClassifier.ClassifyWithContext(ctx, text, history)
		if err == nil && llmResult.Confidence > ruleResult.Confidence {
			return llmResult, nil
		}
	}

	return ruleResult, nil
}

// ClassifyBatch 批量分类
func (hc *HybridIntentClassifier) ClassifyBatch(ctx context.Context, texts []string) ([]*RAGIntent, error) {
	results := make([]*RAGIntent, len(texts))

	for i, text := range texts {
		intent, err := hc.Classify(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = intent
	}

	return results, nil
}

// SetUseLLM 动态开关 LLM
func (hc *HybridIntentClassifier) SetUseLLM(use bool) {
	hc.useLLM = use
}

// SetThresholds 设置置信度阈值
func (hc *HybridIntentClassifier) SetThresholds(high, low float64) {
	hc.highConfidenceThreshold = high
	hc.lowConfidenceThreshold = low
}
