package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino-ext/components/retriever/es8"
	"github.com/cloudwego/eino/components/model"
	er "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/everfid-ever/ThinkForge/core/common"
	"github.com/everfid-ever/ThinkForge/core/config"
	"github.com/everfid-ever/ThinkForge/core/grader"
	"github.com/everfid-ever/ThinkForge/core/indexer"
	"github.com/everfid-ever/ThinkForge/core/retriever"
	"github.com/gogf/gf/v2/frame/g"
)

const (
	scoreThreshold = 1.05 // 文档相似度阈值，用于后续筛选召回结果（防止过低匹配）
	esTopK         = 50   // 检索时最多返回的文档数量
	esTryFindDoc   = 10   // 在部分场景下尝试获取的候选文档数量
)

// Rag 是整个 RAG 系统的核心结构体，封装了检索、索引、问答等组件。
type Rag struct {
	idxer      compose.Runnable[any, []string]                // 同步索引构建器
	idxerAsync compose.Runnable[[]*schema.Document, []string] // 异步索引构建器
	rtrvr      compose.Runnable[string, []*schema.Document]   // 普通文档检索器
	qaRtrvr    compose.Runnable[string, []*schema.Document]   // 问答专用检索器（针对 QA 向量字段）
	client     *elasticsearch.Client                          // Elasticsearch 客户端
	cm         model.BaseChatModel                            // 大语言模型（ChatModel，用于生成答案）

	grader    *grader.Grader // 打分模块暂未启用，开启后会明显降低性能
	conf      *config.Config // 全局配置
	rankScore float64        // 排名分数
}

// New 创建并初始化一个 RAG 核心实例。
// 主要执行：
//  1. 确保 Elasticsearch 索引存在；
//  2. 构建索引器与检索器组件；
//  3. 初始化大语言模型；
func New(ctx context.Context, conf *config.Config) (*Rag, error) {
	if len(conf.IndexName) == 0 {
		return nil, fmt.Errorf("indexName is empty")
	}

	// ① 如果 ES 索引不存在则自动创建
	err := common.CreateIndexIfNotExists(ctx, conf.Client, conf.IndexName)
	if err != nil {
		return nil, err
	}

	// ② 构建索引器（同步）
	buildIndex, err := indexer.BuildIndexer(ctx, conf)
	if err != nil {
		return nil, err
	}

	// ③ 构建索引器（异步，用于批量导入）
	buildIndexAsync, err := indexer.BuildIndexerAsync(ctx, conf)
	if err != nil {
		return nil, err
	}

	// ④ 构建普通文档检索器（Retriever）
	buildRetriever, err := retriever.BuildRetriever(ctx, conf)
	if err != nil {
		return nil, err
	}

	// ⑤ 构建 QA 检索器，检索时使用 QA 向量字段（如 question/answer 嵌入）
	qaCtx := context.WithValue(ctx, common.RetrieverFieldKey, common.FieldQAContentVector)
	qaRetriever, err := retriever.BuildRetriever(qaCtx, conf)
	if err != nil {
		return nil, err
	}

	// ⑥ 初始化聊天模型（大语言模型，如 OpenAI、Claude、Moonshot 等）
	cm, err := common.GetChatModel(ctx, conf.GetChatModelConfig())
	if err != nil {
		g.Log().Error(ctx, "GetChatModel failed, err=%v", err)
		return nil, err
	}

	// ⑦ 返回 RAG 实例
	return &Rag{
		idxer:      buildIndex,
		idxerAsync: buildIndexAsync,
		rtrvr:      buildRetriever,
		qaRtrvr:    qaRetriever,
		client:     conf.Client,
		cm:         cm,
		conf:       conf,
		// grader:  grader.NewGrader(cm), // 暂未启用
	}, nil
}

// GetKnowledgeBaseList 从 Elasticsearch 中获取所有知识库（Knowledge Base）的列表。
// 通过聚合（Aggregation）方式对文档的 knowledge_name 字段去重汇总。
func (x *Rag) GetKnowledgeBaseList(ctx context.Context) (list []string, err error) {
	names := "distinct_knowledge_names"

	// 构建一个 Search Request，只做聚合，不返回文档内容。
	query := search.NewRequest()
	query.Size = common.Of(0) // 不返回实际文档，只做统计
	query.Aggregations = map[string]types.Aggregations{
		names: {
			Terms: &types.TermsAggregation{
				Field: common.Of(common.KnowledgeName), // 按 KnowledgeName 字段分组聚合
				Size:  common.Of(10000),                // 最多返回 10000 个不同知识库名
			},
		},
	}

	// 调用 Elasticsearch 搜索接口执行查询
	res, err := search.NewSearchFunc(x.client)().
		Request(query).
		Do(ctx)
	if err != nil {
		return
	}

	// 若聚合结果为空则直接返回
	if res.Aggregations == nil {
		g.Log().Infof(ctx, "No aggregations found")
		return
	}

	// 解析聚合结果
	termsAgg, ok := res.Aggregations[names].(*types.StringTermsAggregate)
	if !ok || termsAgg == nil {
		err = errors.New("failed to parse terms aggregation")
		return
	}

	// 提取每个分桶的 Key（即知识库名称）
	for _, bucket := range termsAgg.Buckets.([]types.StringTermsBucket) {
		list = append(list, bucket.Key.(string))
	}
	return
}

func (x *Rag) retrieve(ctx context.Context, req *RetrieveReq, qa bool) (msg []*schema.Document, err error) {
	g.Log().Infof(ctx, "query: %v", req.optQuery)
	r := x.rtrvr
	if qa {
		r = x.qaRtrvr
	}
	msg, err = r.Invoke(ctx, req.optQuery,
		compose.WithRetrieverOption(
			er.WithScoreThreshold(req.Score),
			er.WithTopK(req.TopK),
			es8.WithFilters([]types.Query{
				{Match: map[string]types.MatchQuery{common.KnowledgeName: {Query: req.KnowledgeName}}},
			}),
		),
	)
	if err != nil {
		return
	}
	return
}
