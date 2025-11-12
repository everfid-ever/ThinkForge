package common

// 一些在 RAG 或知识库模块中
// 常用的字段名常量定义。
// 这些常量用于统一文档存储、检索、向量化以及扩展信息的字段命名。
const (
	FieldContent         = "content"           // 文档原始内容字段名
	FieldContentVector   = "content_vector"    // 文档内容对应的向量表示字段名
	FieldQAContent       = "qa_content"        // 问答型内容字段名（可能是经过问答对抽取后的内容）
	FieldQAContentVector = "qa_content_vector" // 问答内容对应的向量表示字段名
	FieldExtra           = "ext"               // 扩展字段（用于存放额外的元数据）
	KnowledgeName        = "_knowledge_name"   // 知识库名称字段，用于标识该文档所属的知识库
	DocExtra             = "ext"
	DocQAChunks          = "chunk_id"
	DocQAAnswer          = "answer"

	RetrieverFieldKey = "_retriever_field" // 检索字段标识，用于动态选择检索字段（例如 content_vector 或 qa_content_vector）

	// 标题层级，用于结构化文档内容分级存储或分块索引
	Title1 = "h1" // 一级标题
	Title2 = "h2" // 二级标题
	Title3 = "h3" // 三级标题

	QA_INDEX = "think-forge-qa"
)

var (
	// ExtKeys 定义在 ext（扩展信息）中需要保存的键名。
	// 这些键通常用于描述文档的元信息，如来源、文件名、章节标题等。
	ExtKeys = []string{
		"_extension", // 文件扩展名（例如 .pdf, .docx）
		"_file_name", // 原始文件名
		"_source",    // 文档来源（如网页URL、本地路径）
		Title1,       // 一级标题
		Title2,       // 二级标题
		Title3,       // 三级标题
	}
)
