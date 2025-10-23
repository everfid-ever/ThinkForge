package retriever

import (
	"context"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/everfid-ever/ThinkForge/core/config"
)

// BuildRetriever 构建并编译一个文档检索（Retriever）执行图。
// 该函数主要负责：
//  1. 创建一个检索节点（Retriever Node）；
//  2. 将节点添加到可执行图（Graph）中；
//  3. 定义执行流（从 START → Retriever → END）；
//  4. 编译生成一个可运行的检索组件（Runnable）。
//
// 参数：
//   - ctx：上下文，用于控制生命周期与传递参数。
//   - conf：全局配置，包含 Elasticsearch 客户端、索引名称等信息。
//
// 返回值：
//   - r：编译好的可执行检索组件（Runnable），输入 string 查询文本，输出 []*schema.Document 检索结果。
//   - err：构建或编译过程中出现的错误。
func BuildRetriever(ctx context.Context, conf *config.Config) (r compose.Runnable[string, []*schema.Document], err error) {
	const Retriever = "Retriever" // 定义检索节点的名称常量

	// 创建一个新的执行图（Graph），输入为 string，输出为 []*schema.Document
	g := compose.NewGraph[string, []*schema.Document]()

	// 调用 newRetriever 创建具体的检索器（基于 Elasticsearch 的 Retriever 实例）
	retrieverKeyOfRetriever, err := newRetriever(ctx, conf)
	if err != nil {
		return nil, err
	}

	// 将检索器实例作为一个节点添加到执行图中
	_ = g.AddRetrieverNode(Retriever, retrieverKeyOfRetriever)

	// 定义图的执行顺序：从 START → Retriever → END
	_ = g.AddEdge(compose.START, Retriever)
	_ = g.AddEdge(Retriever, compose.END)

	// 编译执行图，命名为 "rag"
	r, err = g.Compile(ctx, compose.WithGraphName("rag"))
	if err != nil {
		return nil, err
	}

	// 返回编译完成的可运行检索器
	return r, nil
}
