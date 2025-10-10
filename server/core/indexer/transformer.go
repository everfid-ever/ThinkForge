package indexer

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
)

// transformer 是一个复合型文档 Transformer，
// 根据文档类型自动选择对应的分割方式。
//   - Markdown 文档使用 markdown.Splitter
//   - 普通文本使用 recursive.Splitter
type transformer struct {
	markdown  document.Transformer // 处理 Markdown 文档
	recursive document.Transformer // 处理普通文本
}

// newDocumentTransformer 初始化文档分割器（Transformer）组件。
// 该函数作为 RAG 图节点 “DocumentTransformer3” 的初始化逻辑，
// 负责根据文档类型（普通文本 / Markdown）选择不同的分割策略。
//
// 功能概述：
//   - 对普通文本使用递归分割器（Recursive Splitter），将长文档分割为小段；
//   - 对 Markdown 文档使用标题分割器（Markdown Header Splitter），
//     根据标题层级智能拆分段落内容。
//
// 参数：
//   - ctx: 上下文，用于控制超时和取消。
//
// 返回：
//   - document.Transformer: 可自动选择合适分割策略的 Transformer 实例。
//   - error: 初始化过程中出现的错误。
func newDocumentTransformer(ctx context.Context) (tfr document.Transformer, err error) {
	trans := &transformer{}

	// 配置递归分割器
	// 将长文档语义和标点递归拆分，便于后续 Embedding 处理
	config := &recursive.Config{
		ChunkSize:   1000, // 每段内容最多 1000 字
		OverlapSize: 100,  // 段落之间保留 100 字重叠，提高上下文连续性
		Separators:  []string{"\n", ".", "?", "!"},
	}
	recTrans, err := recursive.NewSplitter(ctx, config)
	if err != nil {
		return nil, err
	}

	// 配置 Markdown 分割器
	// 按标题层级（#，##，###）切割文档，保留信息结构
	mdTrans, err := markdown.NewHeaderSplitter(ctx, &markdown.HeaderConfig{
		Headers: map[string]string{
			"#":   "h1", // 一级标题
			"##":  "h2", // 二级标题
			"###": "h3", // 三级标题
		},
		TrimHeaders: false, //保留标题文本
	})
	if err != nil {
		return nil, err
	}

	// 将两种分割器绑定到自定义 transformer
	trans.recursive = mdTrans
	trans.markdown = recTrans

	return trans, nil
}

// Transform 对输入文档执行分割操作。
// 根据文档的扩展名（_extension）判断使用哪种分割策略：
//   - 若是 `.md` 文件，使用 Markdown 分割；
//   - 否则使用递归文本分割。
//
// 参数：
//   - ctx: 上下文，用于控制超时或取消；
//   - docs: 待分割的文档切片；
//   - opts: Transformer 可选参数。
//
// 返回：
//   - []*schema.Document: 分割后的文档列表；
//   - error: 执行过程中产生的错误。
func (x *transformer) Transform(ctx context.Context, docs []*schema.Document, opts ...document.TransformerOption) ([]*schema.Document, error) {
	isMd := false

	// 检查文档是否为 Markdown，仅需要检测第一个文档
	for _, doc := range docs {
		if doc.MetaData["_extension"] == ".md" {
			isMd = true
			break
		}
	}

	// 根据类型选择对应的分割器执行 Transformer
	if isMd {
		return x.markdown.Transform(ctx, docs, opts...)
	}
	return x.recursive.Transform(ctx, docs, opts...)
}
