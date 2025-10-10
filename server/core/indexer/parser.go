package indexer

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/parser/html"
	"github.com/cloudwego/eino-ext/components/document/parser/pdf"
	"github.com/cloudwego/eino/components/document/parser"
	"github.com/everfid-ever/ThinkForge/core/common"
)

// newParser 初始化文档解析器（Parser）组件。
// 它根据文件类型（如 `.html`、`.pdf` 等）自动选择对应的解析器，
// 并在遇到未知格式时回退到默认的文本解析器（TextParser）。
//
// 用途：
//
//	在 RAG（Retrieval-Augmented Generation）流程中，
//	负责将原始文件内容（HTML/PDF/TXT 等）解析为结构化文本，
//	以便后续进行向量化和检索。
//
// 逻辑步骤：
//  1. 初始化默认的文本解析器（textParser）；
//  2. 创建 HTML 解析器并配置选择器；
//  3. 创建 PDF 解析器用于提取 PDF 文本内容；
//  4. 将各类型解析器注册到 ExtParser（扩展解析器）；
//  5. 返回可自动选择解析策略的综合解析器实例。
func newParser(ctx context.Context) (p parser.Parser, err error) {
	// 默认文本解析器，用于处理纯文本文件或未知格式
	textParser := parser.TextParser{}

	// 创建 HTML 解析器，用于从网页内容中提取正文文本。
	// 这里设置选择器为 <body>，表示只解析网页主体内容。
	htmlParser, err := html.NewParser(ctx, &html.Config{
		Selector: common.Of("body"),
	})
	if err != nil {
		return nil, err
	}

	// 创建 PDF 解析器，用于从 PDF 文档中提取可读文本。
	pdfParser, err := pdf.NewPDFParser(ctx, &pdf.Config{})
	if err != nil {
		return
	}

	// 创建“扩展解析器”（ExtParser），将不同格式的解析器统一管理。
	// 它会根据文件扩展名自动选择合适的解析逻辑。
	p, err = parser.NewExtParser(ctx, &parser.ExtParserConfig{
		// 注册特定扩展名对应的解析器
		Parsers: map[string]parser.Parser{
			".html": htmlParser, // 处理 HTML 文件
			".pdf":  pdfParser,  // 处理 PDF 文件
		},
		// 设置默认解析器，用于未知或纯文本格式
		FallbackParser: textParser,
	})
	if err != nil {
		return nil, err
	}

	// 返回最终综合解析器
	return
}
