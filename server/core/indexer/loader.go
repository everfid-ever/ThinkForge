package indexer

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino-ext/components/document/loader/url"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
	"github.com/everfid-ever/ThinkForge/core/common"
)

// multiLoader 是一个自定义文档加载器，
// 同时封装了文件加载器和 URL 加载器。
// 它会根据输入的文档来源自动选择合适的加载方式。
type multiLoader struct {
	fileLoader document.Loader // 文件加载器
	urlLoader  document.Loader // URL 加载器
}

// newLoader 初始化 RAG 图节点中的“Loader1”组件
// 它根据数据来源类型（文件或 URL）构建多来源文档加载器（multiLoader），
// // 用于从本地文件或网页中提取文档内容。
//
// 流程：
// 1. 创建内容解析器（parser）用于提取文档文本；
// 2. 初始化文件加载器（FileLoader），支持从本地文件读取并解析内容；
// 3. 初始化 URL 加载器（UrlLoader），支持从网页链接加载内容；
// 4. 封装成 multiLoader 并返回。
func newLoader(ctx context.Context) (ldr document.Loader, err error) {
	mldr := &multiLoader{}

	// 创建文档解析器（例如用于解析 PDF、TXT、HTML等内容）
	parser, err := newParser(ctx)
	if err != nil {
		return nil, err
	}

	// 创建文档加载器，支持从本地路径加载文件内容
	fldr, err := file.NewFileLoader(ctx, &file.FileLoaderConfig{
		UseNameAsID: false, // 使用文件名作为文档唯一 ID
		Parser:      parser,
	})
	if err != nil {
		return nil, err
	}
	mldr.fileLoader = fldr

	// 创建 URL 加载器，支持从网页连接抓取文档内容
	uldr, err := url.NewLoader(ctx, &url.LoaderConfig{})
	if err != nil {
		return nil, err
	}
	mldr.urlLoader = uldr

	// 返回封装后的多来源加载器

	return mldr, nil
}

// Load 根据文档来源类型选择加载方式。
// 若传入的 URI 是网络地址（URL），则使用 urlLoader；
// 否则，使用 fileLoader 从本地文件加载内容。
//
// 参数：
//   - ctx: 上下文，用于控制取消或超时。
//   - src: 文档来源，包含 URI 等信息。
//   - opts: 加载选项（可选参数）。
//
// 返回值：
//   - []*schema.Document：加载并解析后的文档切片。
//   - error：加载过程中出现的错误。
func (x *multiLoader) Load(ctx context.Context, src document.Source, opts ...document.LoaderOption) ([]*schema.Document, error) {
	// 如果来源是 URL，则使用 URL 加载器
	if common.IsURL(src.URI) {
		return x.urlLoader.Load(ctx, src, opts...)
	}
	// 否则使用文件加载器
	return x.fileLoader.Load(ctx, src, opts...)
}
