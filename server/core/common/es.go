package common

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/indices/create"
	"github.com/elastic/go-elasticsearch/v8/typedapi/indices/exists"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/gogf/gf/v2/frame/g"
)

// createIndex 创建一个新的 Elasticsearch 索引，并定义字段映射 (mappings)。
// 参数：
//   - ctx: 上下文对象，用于控制超时或取消。
//   - client: Elasticsearch 客户端实例。
//   - indexName: 要创建的索引名称。
//
// 功能：
//   - 定义多个字段映射，包括文本字段、关键词字段和向量字段。
//   - 向量字段 (DenseVector) 通常用于语义搜索或向量检索任务。
func createIndex(ctx context.Context, client *elasticsearch.Client, indexName string) error {
	_, err := create.NewCreateFunc(client)(indexName).Request(&create.Request{
		Mappings: &types.TypeMapping{
			Properties: map[string]types.Property{
				// 文本字段：存储文档的主要内容
				FieldContent: types.NewTextProperty(),

				// 扩展字段：存储额外元数据或附加信息
				FieldExtra: types.NewTextProperty(),

				// 知识名称字段：用于关键字检索（不分词）
				KnowledgeName: types.NewKeywordProperty(),

				// 向量字段1：用于存储内容的向量表示（嵌入）
				FieldContentVector: &types.DenseVectorProperty{
					Dims:       Of(1024),     // 向量维度，需与模型一致
					Index:      Of(true),     // 启用向量索引
					Similarity: Of("cosine"), // 余弦相似度检索
				},

				// 向量字段2：用于存储问答内容的向量表示
				FieldQAContentVector: &types.DenseVectorProperty{
					Dims:       Of(1024),
					Index:      Of(true),
					Similarity: Of("cosine"),
				},
			},
		},
	}).Do(ctx)

	return err
}

// CreateIndexIfNotExists 检查指定索引是否存在，不存在则自动创建。
// 参数：
//   - ctx: 上下文对象。
//   - client: Elasticsearch 客户端。
//   - indexName: 索引名称。
//
// 返回：
//   - err: 若操作失败则返回错误。
//
// 逻辑说明：
//   - 使用 exists API 检查索引是否存在。
//   - 若不存在，则调用 createIndex 创建索引。
func CreateIndexIfNotExists(ctx context.Context, client *elasticsearch.Client, indexName string) error {
	indexExists, err := exists.NewExistsFunc(client)(indexName).Do(ctx)
	if err != nil {
		return err
	}
	if indexExists {
		return nil
	}
	err = createIndex(ctx, client, indexName)
	return err
}

// DeleteDocument 删除指定索引中的单个文档。
// 参数：
//   - ctx: 上下文对象。
//   - client: Elasticsearch 客户端。
//   - documentID: 要删除的文档 ID。
//
// 返回：
//   - err: 删除失败时返回错误。
//
// 功能：
//   - 从配置文件读取索引名称（es.indexName）。
//   - 调用 Elasticsearch Delete API 删除指定文档。
//   - 内置重试机制，处理临时网络或服务异常。
func DeleteDocument(ctx context.Context, client *elasticsearch.Client, documentID string) error {
	return withRetry(func() error {
		// 从配置文件中读取索引名称
		indexName := g.Cfg().MustGet(ctx, "es.indexName").String()

		// 调用 Elasticsearch Delete API
		res, err := client.Delete(indexName, documentID)
		if err != nil {
			return fmt.Errorf("delete document failed: %w", err)
		}
		defer res.Body.Close()

		// 判断返回结果是否包含错误
		if res.IsError() {
			return fmt.Errorf("delete document failed: %s", res.String())
		}

		return nil
	})
}

// withRetry 是一个通用的操作重试包装函数。
// 参数：
//   - operation: 要执行的函数（返回 error 表示失败需重试）。
//
// 返回：
//   - err: 若在最大重试时间内操作仍失败，则返回最后一个错误。
//
// 功能：
//   - 使用指数退避 (Exponential Backoff) 策略重试操作。
//   - 适用于网络波动或暂时性错误的场景，如删除、索引或更新操作。
//   - 最大重试持续时间：30 秒。
func withRetry(operation func() error) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 30 * time.Second

	return backoff.Retry(operation, b)
}
