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

// newIndexer 初始化 RAG 系统中的 Elasticsearch 索引器（Indexer）。
//
// 在 RAG（Retrieval-Augmented Generation）流程中，Indexer 的主要职责是：
//   - 将经过解析与切分的文档写入 Elasticsearch 索引；
//   - 同时生成对应的向量嵌入（Embedding），用于语义检索；
//   - 支持通过上下文（context）动态绑定知识库名称，以便区分不同知识域的数据。
//
// 参数：
//
//	ctx  —— 上下文对象，用于传递控制信号和动态元信息（例如知识库名称）。
//	conf —— 配置对象（*config.Config），其中包含：
//	         - Elasticsearch 客户端实例
//	         - 索引名称
//	         - Embedding 模型配置
//
// 返回值：
//
//	idr —— 初始化完成的 Indexer 实例，用于批量写入与向量索引。
//	err —— 初始化过程中的错误（例如连接失败、配置不合法等）。
func newIndexer(ctx context.Context, conf *config.Config) (idr indexer.Indexer, err error) {
	// 构建索引器配置对象
	indexerConfig := &es8.IndexerConfig{
		Client:    conf.Client,    // Elasticsearch 客户端
		Index:     conf.IndexName, // 目标索引名称
		BatchSize: 10,             // 批量写入大小

		// DocumentToFields 定义文档到 ES 字段的映射逻辑
		DocumentToFields: func(ctx context.Context, doc *schema.Document) (field2Value map[string]es8.FieldValue, err error) {
			var knowledgeName string

			// 从 context 中读取知识库名称
			if value, ok := ctx.Value(common.KnowledgeName).(string); ok {
				knowledgeName = value
			} else {
				err = fmt.Errorf("knowledge name not found")
				return
			}

			// 若文档未指定 ID，则自动生成 UUID
			if len(doc.ID) == 0 {
				doc.ID = uuid.New().String()
			}

			// 若文档包含元数据，则序列化拓展信息保存至 "extra" 字段
			if doc.MetaData != nil {
				marshal, _ := sonic.Marshal(doc.MetaData)
				doc.MetaData[common.KnowledgeName] = string(marshal)
			}

			// 返回字段与对应值的映射，用于存储到 ES
			return map[string]es8.FieldValue{
				// 主内容字段（文本＋向量）
				common.FieldContent: {
					Value:    doc.Content,
					EmbedKey: common.FieldContentVector, // 指定该字段需要进行向量化
				},

				// 扩展信息字段（JSON 格式的元数据）
				common.FieldExtra: {
					Value: doc.MetaData[common.FieldExtra],
				},

				// 知识库名称，用于区分不同领域的数据
				common.KnowledgeName: {
					Value: knowledgeName,
				},

				// 如果后续需要对问答内容单独建向量，可启用以下字段
				// common.FieldQAContent: {
				// 	Value:    doc.MetaData[common.FieldQAContent],
				// 	EmbedKey: common.FieldQAContentVector,
				// },
			}, nil
		},
	}

	// 创建文本嵌入实例，用于将内容转换为语义向量
	embeddingIns11, err := common.NewEmbedding(ctx, conf)
	if err != nil {
		return nil, err
	}
	indexerConfig.Embedding = embeddingIns11

	// 基于配置初始化 ES 索引器
	idr, err = es8.NewIndexer(ctx, indexerConfig)
	if err != nil {
		return nil, err
	}

	// 返回可用的索引器实例
	return idr, nil
}
