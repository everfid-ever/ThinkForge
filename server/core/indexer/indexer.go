package indexer

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/components/indexer/es8"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/everfid-ever/ThinkForge/core/common"
	"github.com/everfid-ever/ThinkForge/core/config"
	"github.com/google/uuid"
)

// newIndexer 初始化一个基于 Elasticsearch 8 的向量索引器 (Indexer)。
//
// 功能说明：
// 在 RAG（Retrieval-Augmented Generation）系统中，Indexer 的职责是：
//   - 将文档内容及其元数据（MetaData）写入到 Elasticsearch 索引中；
//   - 将文本内容通过 Embedding 模型转化为语义向量，以支持语义检索；
//   - 根据 context 上下文携带的知识库名称，将文档归类到对应知识域；
//   - 支持批量索引、文档ID自动生成、元数据序列化等特性。
//
// 参数：
//
//   - ctx  : 上下文对象，用于传递请求范围内的控制信号与知识库名称。
//
//   - conf : *config.Config 配置对象，包含以下信息：
//
//   - Elasticsearch 客户端实例
//
//   - 索引名称（IndexName）
//
//   - Embedding 模型的 API Key / BaseURL / 模型名
//
//     返回值：
//
//   - idr : 一个可直接用于索引文档的 Indexer 实例。
//
//   - err : 错误对象（初始化或配置错误时返回）。
func newIndexer(ctx context.Context, conf *config.Config) (idr indexer.Indexer, err error) {
	// 构建 ES 索引器配置
	indexerConfig := &es8.IndexerConfig{
		Client:    conf.Client,    // ES 客户端
		Index:     conf.IndexName, // 索引名称
		BatchSize: 10,             // 批量写入大小（可调优）

		// DocumentToFields: 将 schema.Document 转换为 Elasticsearch 字段映射
		DocumentToFields: func(ctx context.Context, doc *schema.Document) (field2Value map[string]es8.FieldValue, err error) {
			var knowledgeName string

			// 从 context 中提取知识库名称
			if value, ok := ctx.Value(common.KnowledgeName).(string); ok {
				knowledgeName = value
			} else {
				err = fmt.Errorf("knowledge name not found in context")
				return
			}

			// 若文档未指定 ID，则自动生成一个 UUID
			if len(doc.ID) == 0 {
				doc.ID = uuid.New().String()
			}

			// 若存在元数据（MetaData），将其序列化保存至 "ext" 字段
			if doc.MetaData != nil {
				marshal, _ := sonic.Marshal(doc.MetaData)
				doc.MetaData[common.KnowledgeName] = string(marshal)
			}

			// 返回字段与值的映射，用于 ES 索引写入
			return map[string]es8.FieldValue{
				// 主内容字段：用于语义向量检索
				common.FieldContent: {
					Value:    getMdContentWithTitle(doc), // 拼接标题 + 内容
					EmbedKey: common.FieldContentVector,  // 指定向量化键
				},

				// 扩展字段（存储元数据）
				common.FieldExtra: {
					Value: doc.MetaData[common.FieldExtra],
				},

				// 知识库名称字段（用于逻辑隔离）
				common.KnowledgeName: {
					Value: knowledgeName,
				},

				// 可选：问答内容字段（如需对 QA 对进行单独向量化，可启用）
				// common.FieldQAContent: {
				// 	Value:    doc.MetaData[common.FieldQAContent],
				// 	EmbedKey: common.FieldQAContentVector,
				// },
			}, nil
		},
	}

	// 创建 Embedding 实例（用于将文本内容转化为向量）
	embeddingIns, err := common.NewEmbedding(ctx, conf)
	if err != nil {
		return nil, fmt.Errorf("create embedding instance failed: %w", err)
	}
	indexerConfig.Embedding = embeddingIns

	// 初始化 ES8 索引器
	idr, err = es8.NewIndexer(ctx, indexerConfig)
	if err != nil {
		return nil, fmt.Errorf("init es8 indexer failed: %w", err)
	}

	return idr, nil
}

// getMdContentWithTitle 拼接文档的标题与正文内容。
//
// 说明：
//   - 若文档 MetaData 中包含标题信息（h1~h6），则会将其附加在正文前；
//   - 若未检测到标题，则仅返回文档内容。
//   - 该设计可提高文本语义的完整性与向量检索效果。
func getMdContentWithTitle(doc *schema.Document) string {
	if doc.MetaData == nil {
		return doc.Content
	}

	title := ""
	list := []string{"h1", "h2", "h3", "h4", "h5", "h6"}

	// 依次拼接标题层级
	for _, v := range list {
		if d, ok := doc.MetaData[v].(string); ok && len(d) > 0 {
			title += fmt.Sprintf("%s: %s ", v, d)
		}
	}

	if len(title) == 0 {
		return doc.Content
	}

	return title + "\n" + doc.Content
}

func getExtData(doc *schema.Document) map[string]any {
	if doc.MetaData == nil {
		return nil
	}
	res := make(map[string]any)
	for _, key := range common.ExtKeys {
		if v, e := doc.MetaData[key]; e {
			res[key] = v
		}
	}
	return res
}
