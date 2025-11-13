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
					Value:    doc.Content,               // 文档内容
					EmbedKey: common.FieldContentVector, // 指定向量化键
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
