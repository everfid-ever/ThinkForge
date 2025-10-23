package retriever

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/components/retriever/es8"
	"github.com/cloudwego/eino-ext/components/retriever/es8/search_mode"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/everfid-ever/ThinkForge/core/common"
	"github.com/everfid-ever/ThinkForge/core/config"
)

// newRetriever 组件初始化函数
// 此函数用于创建一个基于 Elasticsearch 8 的向量检索器（Retriever）。
// 它会根据配置初始化检索参数（包括索引、搜索模式、embedding 模型等），
// 用于在知识库中执行语义相似度搜索。
//
// 节点名称：'Retriever1' in graph 'rag'
//
// 参数：
//
//	ctx  - 上下文，用于控制超时、取消及传递自定义检索参数
//	conf - 全局配置对象，包含 ES 客户端、索引名、API Key、模型名称等
//
// 返回：
//
//	rtr - 初始化完成的 Retriever（语义检索组件）
//	err - 初始化失败时返回错误
func newRetriever(ctx context.Context, conf *config.Config) (rtr retriever.Retriever, err error) {
	// 默认向量字段为内容向量
	vectorField := common.FieldContentVector

	// 如果在 content 中传入特定的向量字段（如 QA向量），则优先使用
	if value, ok := ctx.Value(common.RetrieverFieldKey).(string); ok {
		vectorField = value
	}

	// 构建 ES 8 检索器配置
	retrieverConfig := &es8.RetrieverConfig{
		Client: conf.Client, // ES 客户端实例
		Index:  conf.IndexName,
		// 指定搜索模式：基于密集向量相似度
		SearchMode: search_mode.SearchModeDenseVectorSimilarity(
			search_mode.DenseVectorSimilarityTypeCosineSimilarity, // 使用余弦相似度
			vectorField,                                           // 向量字段名
		),
		// 指定搜索结果的解析函数（将 ES 命中结果转为 schema.Document）
		ResultParser: EsHit2Document,
	}

	// 创建 embedding 实例，用于将查询文本转为向量
	embeddingIns11, err := common.NewEmbedding(ctx, conf)
	if err != nil {
		return nil, err
	}
	retrieverConfig.Embedding = embeddingIns11

	// 构建 Retriever 组件
	rtr, err = es8.NewRetriever(ctx, retrieverConfig)
	if err != nil {
		return nil, err
	}

	return rtr, nil
}

// EsHit2Document 将 Elasticsearch 的命中结果（Hit）转换为通用文档结构（schema.Document）
// 在 Retriever 查询时，每条命中结果（Hit）都会被传入该函数进行转换。
// 函数负责：
//  1. 解析文档 ID 与内容
//  2. 读取密集向量（Dense Vector）
//  3. 提取元数据（MetaData）
//  4. 保留打分信息（Score）
//
// 参数：
//
//	ctx - 上下文
//	hit - ES 返回的一条命中记录
//
// 返回：
//
//	doc - 转换后的文档对象（包含内容、元数据、向量等）
//	err - 转换过程中可能发生的错误
func EsHit2Document(ctx context.Context, hit types.Hit) (doc *schema.Document, err error) {
	// 初始化基础文档对象
	doc = &schema.Document{
		ID:       *hit.Id_,         // ES 文档 ID
		MetaData: map[string]any{}, // 初始化空元数据
	}

	// 解析原始 JSON 源数据（_source）
	var src map[string]any
	if err = sonic.Unmarshal(hit.Source_, &src); err != nil {
		return nil, err
	}

	// 遍历 ES 文档的字段，根据类型填充 Document 对象
	for field, val := range src {
		switch field {
		case common.FieldContent:
			// 文档正文内容
			doc.Content = val.(string)

		case common.FieldContentVector:
			// 向量字段（dense vector），将 interface{} 转为 []float64
			var v []float64
			for _, item := range val.([]interface{}) {
				v = append(v, item.(float64))
			}
			doc.WithDenseVector(v)

		case common.FieldQAContentVector, common.FieldQAContent:
			// QA 字段（问答内容与向量）不返回给调用方，直接跳过

		case common.FieldExtra:
			// 元数据扩展字段（例如来源信息、文件名等）
			if val == nil {
				continue
			}
			doc.MetaData[common.FieldExtra] = val.(string)

		case common.KnowledgeName:
			// 所属知识库名称
			doc.MetaData[common.KnowledgeName] = val.(string)

		default:
			// 发现未定义字段，返回错误方便调试
			return nil, fmt.Errorf("unexpected field=%s, val=%v", field, val)
		}
	}

	// 如果有打分信息（ES 返回 _score 字段），附加到文档
	if hit.Score_ != nil {
		doc.WithScore(float64(*hit.Score_))
	}

	return doc, nil
}
