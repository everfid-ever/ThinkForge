package rag

import (
	"context"
	"github.com/everfid-ever/ThinkForge/core"

	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/internal/logic/rag"
)

// Indexer 处理文档入库（索引）请求。
//
// @Summary 文档索引接口（支持文件上传或网络URL）
// @Description 将用户提供的文件或网络文档解析为文本，并进行向量化索引，存储到指定知识库。
// @Tags rag
// @Accept multipart/form-data
// @Produce application/json
// @Param file formData file false "本地上传文件（可选）"
// @Param url formData string false "网络文件地址（可选）"
// @Param knowledge_name formData string true "知识库名称"
// @Success 200 {object} v1.IndexerRes "索引成功后返回文档ID列表"
// @Failure 400 {object} ghttp.DefaultHandlerResponse "参数错误或上传失败"
// @Router /v1/indexer [post]
func (c *ControllerV1) Indexer(ctx context.Context, req *v1.IndexerReq) (res *v1.IndexerRes, err error) {
	// 获取 RAG 服务实例（封装了索引逻辑，如分词、嵌入、存储等）
	svr := rag.GetRagSvr()

	// uri 用于存储待索引的文档路径（可能来自 URL 或上传文件）
	uri := req.URL

	// 如果用户上传了本地文件，则优先使用文件路径
	if req.File != nil {
		// 将上传的文件保存到服务器本地 uploads 目录下
		filename, e := req.File.Save("./uploads/")
		if e != nil {
			// 若保存失败，则返回错误
			err = e
			return
		}
		// 生成本地文件完整路径（后续会传给 RAG 服务）
		uri = "./uploads/" + filename
	}

	// 调用 RAG 核心逻辑执行文档索引
	indexReq := &core.IndexReq{
		URI:           uri,
		KnowledgeName: req.KnowledgeName,
	}
	ids, err := svr.Index(ctx, indexReq)
	if err != nil {
		// 索引失败，直接返回错误
		return
	}

	// 封装返回结果，包含所有被成功索引的文档 ID
	res = &v1.IndexerRes{
		DocIDs: ids,
	}
	return
}
