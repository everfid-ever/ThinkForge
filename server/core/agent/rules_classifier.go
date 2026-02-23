package agent

import (
	"context"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// 权重定义
	kwWeight      = 1.0 // 普通关键词权重
	hotWordWeight = 2.0 // 强信号关键词权重
	patternWeight = 3.0 // 正则模式权重
	domainWeight  = 1.5 // 领域关键词权重

	// 阈值定义
	minConfidenceThreshold  = 0.3 // 最低置信度
	highConfidenceThreshold = 0.7 // 高置信度（无需LLM）
)

// IntentRule 意图规则
type IntentRule struct {
	IntentType     RAGIntentType
	Keywords       []string         // 普通关键词
	HotWords       []string         // 强信号关键词
	Patterns       []*regexp.Regexp // 正则模式
	DomainKeywords []string         // 领域关键词
	Weight         float64          // 规则权重

	// 策略提示
	SuggestedStrategy string   // 推荐策略
	SuggestedTools    []string // 推荐工具
	EstimatedSteps    int      // 预估步数
}

// RuleBasedClassifier 基于规则的分��器
type RuleBasedClassifier struct {
	mu          sync.RWMutex
	rules       []IntentRule
	slotParsers []SlotParser
}

// SlotParser 槽位解析器
type SlotParser struct {
	SlotName string
	Pattern  *regexp.Regexp
}

// NewRuleBasedClassifier 创建规则分类器
func NewRuleBasedClassifier() *RuleBasedClassifier {
	rc := &RuleBasedClassifier{}
	rc.initRAGRules()
	rc.initSlotParsers()
	return rc
}

func (rc *RuleBasedClassifier) initRAGRules() {
	rules := []IntentRule{
		// ===== 简单问答 =====
		{
			IntentType:        RAGIntentSimpleQA,
			Keywords:          []string{"什么是", "定义", "explain", "define", "介绍", "含义"},
			HotWords:          []string{"是什么", "指的是", "意思是"},
			Patterns:          mustCompilePatterns(`(?i)^(what is|define|explain)\s+\w+\??$`, `^什么是[\u4e00-\u9fa5]+[？?]?$`),
			Weight:            1.0,
			SuggestedStrategy: "simple_rag",
			SuggestedTools:    []string{"rag"},
			EstimatedSteps:    1,
		},

		// ===== 事实核查 =====
		{
			IntentType:        RAGIntentFactCheck,
			Keywords:          []string{"是否", "真的", "确认", "verify", "check", "是真的吗"},
			HotWords:          []string{"是不是", "对不对", "有没有"},
			Patterns:          mustCompilePatterns(`(?i)(is it true|verify|confirm)`, `是(真的|假的|对的|错的)`),
			Weight:            1.1,
			SuggestedStrategy: "simple_rag",
			SuggestedTools:    []string{"rag"},
			EstimatedSteps:    2,
		},

		// ===== 多跳推理 =====
		{
			IntentType:        RAGIntentMultiHopQA,
			Keywords:          []string{"为什么", "原因", "如何", "怎么", "影响", "导致", "关系"},
			HotWords:          []string{"背后的原因", "如何实现", "工作原理", "为什么会"},
			Patterns:          mustCompilePatterns(`(为什么|why).*(导致|影响|实现|会)`, `(如何|how).*(实现|工作|运行)`),
			DomainKeywords:    []string{"原理", "机制", "过程"},
			Weight:            1.2,
			SuggestedStrategy: "react_agent",
			SuggestedTools:    []string{"rag"},
			EstimatedSteps:    3,
		},

		// ===== 因果推理 =====
		{
			IntentType:        RAGIntentCausalReasoning,
			Keywords:          []string{"因为", "所以", "导致", "造成", "引起", "产生"},
			HotWords:          []string{"根本原因", "直接原因", "间接影响"},
			Patterns:          mustCompilePatterns(`(因为|because).*(所以|therefore)`, `(导致|cause|lead to).*(结果|result)`),
			Weight:            1.3,
			SuggestedStrategy: "react_agent",
			SuggestedTools:    []string{"rag"},
			EstimatedSteps:    4,
		},

		// ===== 过程性问题 =====
		{
			IntentType:        RAGIntentProcedural,
			Keywords:          []string{"步骤", "如何做", "怎么做", "流程", "操作", "教程"},
			HotWords:          []string{"一步一步", "详细步骤", "操作指南"},
			Patterns:          mustCompilePatterns(`(如何|怎么)(做|操作|实现|配置)`, `(step by step|how to)`),
			Weight:            1.1,
			SuggestedStrategy: "simple_rag",
			SuggestedTools:    []string{"rag"},
			EstimatedSteps:    2,
		},

		// ===== 对比分析 =====
		{
			IntentType:        RAGIntentComparison,
			Keywords:          []string{"对比", "比较", "区别", "差异", "compare", "difference", "vs", "versus"},
			HotWords:          []string{"哪个更好", "优缺点", "选择哪个", "异同点"},
			Patterns:          mustCompilePatterns(`(对比|比较|compare).*(和|与|vs|versus)`, `\w+\s+(vs|versus)\s+\w+`, `(优缺点|pros and cons)`),
			Weight:            1.3,
			SuggestedStrategy: "react_agent",
			SuggestedTools:    []string{"rag"},
			EstimatedSteps:    4,
		},

		// ===== 摘要生成 =====
		{
			IntentType:        RAGIntentSummarization,
			Keywords:          []string{"总结", "概括", "summarize", "摘要", "归纳", "概述"},
			HotWords:          []string{"用一句话", "简要说明", "核心内容"},
			Patterns:          mustCompilePatterns(`(总结|summarize|概括).*(所有|全部|整个)`, `简要(说明|介绍|描述)`),
			Weight:            1.0,
			SuggestedStrategy: "simple_rag",
			SuggestedTools:    []string{"rag"},
			EstimatedSteps:    2,
		},

		// ===== 数据聚合 =====
		{
			IntentType:        RAGIntentAggregation,
			Keywords:          []string{"统计", "计算", "总共", "平均", "最大", "最小", "多少"},
			HotWords:          []string{"一共有", "总数", "数量"},
			Patterns:          mustCompilePatterns(`(统计|计算|count|sum).*(数量|总数|平均)`, `(有多少|how many)`),
			Weight:            1.4,
			SuggestedStrategy: "react_agent",
			SuggestedTools:    []string{"rag", "calculator"},
			EstimatedSteps:    3,
		},

		// ===== 趋势分析 =====
		{
			IntentType:        RAGIntentTrendAnalysis,
			Keywords:          []string{"趋势", "变化", "增长", "下降", "发展", "演变"},
			HotWords:          []string{"发展趋势", "变化趋势", "未来走向"},
			Patterns:          mustCompilePatterns(`(趋势|trend|变化|change).*(分析|analysis)`, `(��长|下降).*(率|速度)`),
			Weight:            1.3,
			SuggestedStrategy: "react_agent",
			SuggestedTools:    []string{"rag", "calculator"},
			EstimatedSteps:    4,
		},

		// ===== 混合检索 =====
		{
			IntentType:        RAGIntentHybridSearch,
			Keywords:          []string{"最新", "最近", "当前", "现在", "latest", "current", "recent"},
			HotWords:          []string{"最新进展", "当前状态", "最近发生"},
			Patterns:          mustCompilePatterns(`(最新|latest|最近|recent).*(消息|进展|状态|新闻)`, `(当前|current|现在|now)`),
			Weight:            1.4,
			SuggestedStrategy: "hybrid",
			SuggestedTools:    []string{"rag", "web_search"},
			EstimatedSteps:    3,
		},

		// ===== 实时查询 =====
		{
			IntentType:        RAGIntentRealtimeQuery,
			Keywords:          []string{"今天", "昨天", "明天", "现在", "实时", "当前"},
			HotWords:          []string{"实时数据", "最新数据", "当前值"},
			Patterns:          mustCompilePatterns(`(今天|昨天|明天|today|yesterday|tomorrow)`, `(实时|real-time|即时)`),
			Weight:            1.5,
			SuggestedStrategy: "hybrid",
			SuggestedTools:    []string{"rag", "web_search", "database"},
			EstimatedSteps:    2,
		},

		// ===== 代码生成 =====
		{
			IntentType:        RAGIntentCodeGeneration,
			Keywords:          []string{"代码", "实现", "code", "implement", "写一个", "生成"},
			HotWords:          []string{"写代码", "代码示例", "实现代码"},
			Patterns:          mustCompilePatterns(`(写|生成|create).*(代码|code)`, `(implement|实现).*(function|函数|方法)`),
			DomainKeywords:    []string{"python", "go", "java", "javascript", "function", "class"},
			Weight:            1.2,
			SuggestedStrategy: "react_agent",
			SuggestedTools:    []string{"rag", "code_executor"},
			EstimatedSteps:    3,
		},

		// ===== 内容创作 =====
		{
			IntentType:        RAGIntentContentCreation,
			Keywords:          []string{"写", "创作", "生成", "制作", "设计"},
			HotWords:          []string{"帮我写", "帮我生成", "创作一个"},
			Patterns:          mustCompilePatterns(`(写|创作|生成|create).*(文章|方案|报告|计划)`, `(帮我|help me).*(写|生成|create)`),
			Weight:            1.1,
			SuggestedStrategy: "react_agent",
			SuggestedTools:    []string{"rag"},
			EstimatedSteps:    4,
		},

		// ===== 问题澄清 =====
		{
			IntentType:        RAGIntentClarification,
			Keywords:          []string{"不太明白", "什么意思", "能详细", "具体", "再说一遍"},
			HotWords:          []string{"不太理解", "没听懂", "解释一下"},
			Patterns:          mustCompilePatterns(`(不(太)?(明白|理解|懂)|what do you mean)`, `(详细|具体|详细说明)`),
			Weight:            0.9,
			SuggestedStrategy: "simple_rag",
			SuggestedTools:    []string{"rag"},
			EstimatedSteps:    1,
		},
	}

	// 预处理：关键词转小写
	for i := range rules {
		for j, kw := range rules[i].Keywords {
			rules[i].Keywords[j] = strings.ToLower(kw)
		}
		for j, hw := range rules[i].HotWords {
			rules[i].HotWords[j] = strings.ToLower(hw)
		}
		for j, dk := range rules[i].DomainKeywords {
			rules[i].DomainKeywords[j] = strings.ToLower(dk)
		}
	}

	rc.rules = rules
}

func (rc *RuleBasedClassifier) initSlotParsers() {
	rc.slotParsers = []SlotParser{
		{
			SlotName: "time",
			Pattern:  regexp.MustCompile(`\d{4}[-/年]\d{1,2}[-/月]\d{1,2}[日]?|\d{1,2}[-/月]\d{1,2}[日]?|今天|明天|昨天|上周|本月|去年|last week|yesterday|today|tomorrow|\d+\s*(minutes?|hours?|days?|weeks?|months?|years?)\s*(ago|later|前|后)`),
		},
		{
			SlotName: "number",
			Pattern:  regexp.MustCompile(`\d+(?:\.\d+)?|[一二三四五六七八九十百千万亿]+`),
		},
		{
			SlotName: "entity",
			Pattern:  regexp.MustCompile(`[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*|[\u4e00-\u9fa5]{2,}`),
		},
	}
}

// Classify 分类意图
func (rc *RuleBasedClassifier) Classify(ctx context.Context, text string) (*RAGIntent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rc.mu.RLock()
	defer rc.mu.RUnlock()

	normalized := rc.normalize(text)
	scores := rc.score(normalized)
	slots := rc.extractSlots(text)

	// 按分数排序
	var candidates []struct {
		intentType RAGIntentType
		score      float64
		rule       *IntentRule
	}

	for intentType, score := range scores {
		if score < minConfidenceThreshold {
			continue
		}

		// 找到对应的规则
		for i := range rc.rules {
			if rc.rules[i].IntentType == intentType {
				candidates = append(candidates, struct {
					intentType RAGIntentType
					score      float64
					rule       *IntentRule
				}{
					intentType: intentType,
					score:      score,
					rule:       &rc.rules[i],
				})
				break
			}
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// 构建意图结果
	if len(candidates) == 0 {
		return &RAGIntent{
			Type:                 RAGIntentUnknown,
			Confidence:           0.0,
			RawText:              text,
			Strategy:             "simple_rag",
			NeedTools:            []string{"rag"},
			EstimatedSteps:       1,
			Complexity:           ComplexitySimple,
			ClassificationMethod: "rule",
			Timestamp:            time.Now().Format(time.RFC3339),
		}, nil
	}

	best := candidates[0]
	complexity := estimateComplexity(text, best.rule.EstimatedSteps)

	intent := &RAGIntent{
		Type:                 best.intentType,
		Confidence:           best.score,
		RawText:              text,
		Strategy:             best.rule.SuggestedStrategy,
		NeedTools:            best.rule.SuggestedTools,
		EstimatedSteps:       best.rule.EstimatedSteps,
		Complexity:           complexity,
		RequiresExternal:     requiresExternal(best.intentType, text),
		KnowledgeDomains:     extractDomains(text, slots),
		ClassificationMethod: "rule",
		Timestamp:            time.Now().Format(time.RFC3339),
	}

	// 提取时间和范围约束
	intent.TimeConstraint = extractTimeConstraint(text, slots)
	intent.ScopeConstraint = extractScopeConstraint(text, slots)

	return intent, nil
}

// score 计算每个意图的得分（改进版）
func (rc *RuleBasedClassifier) score(text string) map[RAGIntentType]float64 {
	scores := make(map[RAGIntentType]float64)
	lower := strings.ToLower(text)

	for _, rule := range rc.rules {
		var raw float64
		var hitCount, totalSignals int

		// 关键词匹配
		totalSignals += len(rule.Keywords)
		for _, kw := range rule.Keywords {
			if strings.Contains(lower, kw) {
				raw += kwWeight
				hitCount++
			}
		}

		// 强信号词匹配
		totalSignals += len(rule.HotWords)
		for _, hw := range rule.HotWords {
			if strings.Contains(lower, hw) {
				raw += hotWordWeight
				hitCount++
			}
		}

		// 正则匹配
		totalSignals += len(rule.Patterns)
		for _, pattern := range rule.Patterns {
			if pattern.MatchString(text) {
				raw += patternWeight
				hitCount++
			}
		}

		// 领域关键词匹配
		totalSignals += len(rule.DomainKeywords)
		for _, dk := range rule.DomainKeywords {
			if strings.Contains(lower, dk) {
				raw += domainWeight
				hitCount++
			}
		}

		if raw <= 0 || totalSignals == 0 {
			continue
		}

		// 改进的评分函数：结合命中率和原始分数
		hitRate := float64(hitCount) / float64(totalSignals)

		// 饱和函数：避免过度惩罚低命中率
		saturation := 1.0 - math.Exp(-3.0*hitRate)

		// 对数归一化
		maxRaw := (float64(len(rule.Keywords))*kwWeight +
			float64(len(rule.HotWords))*hotWordWeight +
			float64(len(rule.Patterns))*patternWeight +
			float64(len(rule.DomainKeywords))*domainWeight) * rule.Weight

		logNorm := math.Log1p(raw*rule.Weight) / math.Log1p(maxRaw)

		// 加权组合
		finalScore := 0.6*logNorm + 0.4*saturation

		if finalScore > scores[rule.IntentType] {
			scores[rule.IntentType] = finalScore
		}
	}

	return scores
}

// ClassifyWithContext 带上下文的分类
func (rc *RuleBasedClassifier) ClassifyWithContext(ctx context.Context, text string, history []string) (*RAGIntent, error) {
	// 简化版：先实现基础分类，后续可增强上下文理解
	return rc.Classify(ctx, text)
}

// ClassifyBatch 批量分类
func (rc *RuleBasedClassifier) ClassifyBatch(ctx context.Context, texts []string) ([]*RAGIntent, error) {
	results := make([]*RAGIntent, len(texts))
	for i, text := range texts {
		intent, err := rc.Classify(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = intent
	}
	return results, nil
}

// 辅助函数
func (rc *RuleBasedClassifier) normalize(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Join(strings.Fields(text), " ")
	text = fullWidthToHalf(text)
	return text
}

func (rc *RuleBasedClassifier) extractSlots(text string) map[string][]string {
	slots := make(map[string][]string)
	for _, parser := range rc.slotParsers {
		if matches := parser.Pattern.FindAllString(text, -1); len(matches) > 0 {
			slots[parser.SlotName] = matches
		}
	}
	return slots
}

func mustCompilePatterns(patterns ...string) []*regexp.Regexp {
	result := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		result = append(result, regexp.MustCompile(p))
	}
	return result
}

func fullWidthToHalf(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= '！' && r <= '～':
			b.WriteRune(r - 0xFEE0)
		case r == '　':
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func estimateComplexity(text string, steps int) ComplexityLevel {
	if steps <= 2 && len(text) < 30 {
		return ComplexitySimple
	}
	if steps >= 5 || len(text) > 100 {
		return ComplexityComplex
	}
	return ComplexityMedium
}

func requiresExternal(intentType RAGIntentType, text string) bool {
	if intentType == RAGIntentHybridSearch || intentType == RAGIntentRealtimeQuery {
		return true
	}

	externalKeywords := []string{"最新", "实时", "latest", "current", "今天", "昨天", "now"}
	lower := strings.ToLower(text)
	for _, kw := range externalKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	return false
}

func extractDomains(text string, slots map[string][]string) []string {
	// 简化版：基于关键词识别领域
	domains := []string{}
	domainMap := map[string][]string{
		"machine_learning": {"机器学习", "深度学习", "神经网络", "模型", "训练"},
		"database":         {"数据库", "SQL", "查询", "索引", "事务"},
		"web_development":  {"前端", "后端", "API", "接口", "框架"},
		"devops":           {"运维", "部署", "容器", "k8s", "docker"},
	}

	lower := strings.ToLower(text)
	for domain, keywords := range domainMap {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				domains = append(domains, domain)
				break
			}
		}
	}

	return domains
}

func extractTimeConstraint(text string, slots map[string][]string) *TimeConstraint {
	if times, ok := slots["time"]; ok && len(times) > 0 {
		return &TimeConstraint{
			Relative: times[0],
		}
	}
	return nil
}

func extractScopeConstraint(text string, slots map[string][]string) *ScopeConstraint {
	if entities, ok := slots["entity"]; ok && len(entities) > 0 {
		return &ScopeConstraint{
			Entities: entities,
		}
	}
	return nil
}
