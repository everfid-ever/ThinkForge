package rag

import (
	"context"
	"github.com/everfid-ever/ThinkForge/core"
	"github.com/everfid-ever/ThinkForge/internal/logic/knowledge"
	"github.com/everfid-ever/ThinkForge/internal/model/entity"
	"github.com/gogf/gf/v2/frame/g"

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
	svr := rag.GetRagSvr()
	uri := req.URL
	if req.File != nil {
		filename, e := req.File.Save("./uploads/")
		if e != nil {
			err = e
			return
		}
		uri = "./uploads/" + filename
	}

	documents := entity.KnowledgeDocuments{
		KnowledgeBaseName: req.KnowledgeName,
		FileName:          req.File.Filename,
		Status:            int(v1.StatusPending),
	}
	documentsId, err := knowledge.SaveDocumentsInfo(ctx, documents)
	if err != nil {
		g.Log().Errorf(ctx, "SaveDocumentsInfo failed, err=%v", err)
		return
	}

	indexReq := &core.IndexReq{
		URI:           uri,
		KnowledgeName: req.KnowledgeName,
		DocumentsId:   documentsId,
	}
	ids, err := svr.Index(ctx, indexReq)
	if err != nil {
		return
	}
	res = &v1.IndexerRes{
		DocIDs: ids,
	}
	return
}
