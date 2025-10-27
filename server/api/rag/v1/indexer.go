package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// IndexerReq 定义了 /v1/indexer 接口的请求参数。
// 用于向 RAG 系统提交文件（本地上传或网络 URL）以进行向量化索引。
// 索引完成后，可在知识库中进行检索问答。
type IndexerReq struct {
	g.Meta `path:"/v1/indexer" method:"post" mime:"multipart/form-data" tags:"rag"`

	// File 表示上传的本地文件。
	// 前端应使用 multipart/form-data 格式提交。
	// 示例：curl -X POST -F "file=@document.pdf" -F "knowledge_name=MyKB" http://localhost:8080/v1/indexer
	File *ghttp.UploadFile `p:"file" type:"file" dc:"本地文件上传（可选）"`

	// URL 表示网络文件地址。
	// 若文件已在网络上（如OSS、GitHub、公共文档链接），只需填写 URL 即可。
	// 示例：curl -X POST -F "url=https://example.com/doc.pdf" -F "knowledge_name=MyKB" http://localhost:8080/v1/indexer
	URL string `p:"url" dc:"网络文件地址（可选）"`

	// KnowledgeName 表示要索引到的知识库名称。
	// RAG 系统会将该文件的向量数据存储到对应知识库中。
	// 必填参数。
	KnowledgeName string `p:"knowledge_name" dc:"知识库名称" v:"required"`
}

// IndexerRes 定义了 /v1/indexer 接口的响应数据。
// 当文件被成功解析并索引后，系统会返回生成的文档 ID 列表。
type IndexerRes struct {
	g.Meta `mime:"application/json"`

	// DocIDs 表示索引成功的文档唯一标识列表。
	// 每个文档 ID 对应 ES 或向量数据库中的一条记录，可用于后续查询。
	DocIDs []string `json:"doc_ids"`
}
