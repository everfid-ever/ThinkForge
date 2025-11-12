package indexer

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/everfid-ever/ThinkForge/core/config"
)

// BuildIndexer 构建文档索引处理流程（Indexing Pipeline）
// 该函数基于 compose.Graph 组合多个数据处理节点，
// 实现从“加载文档”到“切分内容”、“添加文档ID与合并元数据”再到“索引入库”的完整链路。
//
// 整体数据流：
// [START] → Loader → DocumentTransformer → DocAddIDAndMerge → Indexer → [END]
//
// 参数：
//
//	ctx  - 上下文（控制超时与取消）
//	conf - 配置信息（包括 ES 客户端、模型配置等）
//
// 返回：
//
//	r   - 已编译的可运行流程（compose.Runnable）
//	err - 构建失败时的错误信息
func BuildIndexer(ctx context.Context, conf *config.Config) (r compose.Runnable[any, []string], err error) {
	const (
		Loader1              = "Loader"              // 文档加载节点
		Indexer2             = "Indexer"             // 向量索引节点
		DocumentTransformer3 = "DocumentTransformer" // 文档切分节点
		DocAddIDAndMerge     = "DocAddIDAndMerge"    // 文档ID添加与元数据合并节点
		// QA                   = "QA"               // （可选）问答生成节点（目前注释掉）
	)

	// 创建一个新的可组合图（graph），用于管理节点和数据流转关系
	g := compose.NewGraph[any, []string]()

	// 1️. 初始化 Loader 节点 —— 负责加载文件或URL来源的文档内容
	loader1KeyOfLoader, err := newLoader(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddLoaderNode(Loader1, loader1KeyOfLoader)

	// 2️. 初始化 Indexer 节点 —— 负责将文本内容向量化后写入 Elasticsearch
	indexer2KeyOfIndexer, err := newAsyncIndexer(ctx, conf)
	if err != nil {
		return nil, err
	}
	_ = g.AddIndexerNode(Indexer2, indexer2KeyOfIndexer)

	// 3️. 初始化 DocumentTransformer 节点 —— 负责将文档按段落/语义块切分
	documentTransformer2KeyOfDocumentTransformer, err := newDocumentTransformer(ctx)
	if err != nil {
		return nil, err
	}

	_ = g.AddLambdaNode(DocAddIDAndMerge, compose.InvokableLambda(docAddIDAndMerge))
	// 4️. 添加 Lambda 节点 —— 负责为文档添加唯一ID，并合并元数据
	// （Lambda 节点即执行一个自定义函数）
	//_ = g.AddLambdaNode(DocAddIDAndMerge, compose.InvokableLambda(docAddIDAndMerge))

	// （可选）问答生成节点 QA，目前暂未启用，可在后续扩展为异步内容生成
	// _ = g.AddLambdaNode(QA, compose.InvokableLambda(qa))

	// 5️. 将文档切分节点加入图
	_ = g.AddDocumentTransformerNode(DocumentTransformer3, documentTransformer2KeyOfDocumentTransformer)

	// 6️. 定义节点依赖关系（数据流路径）
	_ = g.AddEdge(compose.START, Loader1)                 // 流程起点 → 加载文档
	_ = g.AddEdge(Loader1, DocumentTransformer3)          // 文档加载 → 文档切分
	_ = g.AddEdge(DocumentTransformer3, DocAddIDAndMerge) // 切分结果 → 添加ID
	// _ = g.AddEdge(DocAddIDAndMerge, QA)                // （可选）可扩展异步 QA 节点
	// _ = g.AddEdge(QA, Indexer2)
	_ = g.AddEdge(DocAddIDAndMerge, Indexer2)
	_ = g.AddEdge(Indexer2, compose.END) // 索引完成 → 流程结束

	// 7️. 编译图为可执行的 Pipeline
	r, err = g.Compile(ctx, compose.WithGraphName("indexer"))
	if err != nil {
		return nil, err
	}

	return r, err
}
