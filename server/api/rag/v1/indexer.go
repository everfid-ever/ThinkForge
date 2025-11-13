package v1

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type IndexerReq struct {
	g.Meta        `path:"/v1/indexer" method:"post" mime:"multipart/form-data" tags:"rag"`
	File          *ghttp.UploadFile `p:"file" type:"file" dc:"If it's a local file, upload the file directly."`
	URL           string            `p:"url" dc:"If it's a network file, just enter the URL."`
	KnowledgeName string            `p:"knowledge_name" dc:"knowledge base name" v:"required"`
}

type IndexerRes struct {
	g.Meta `mime:"application/json"`
	DocIDs []string `json:"doc_ids"`
}
