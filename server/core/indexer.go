package core

import (
	"context"
	"github.com/cloudwego/eino/schema"
)

type IndexReq struct {
	URI           string // 文档地址，可以是文件路径（pdf，html，md等），也可以是网址
	KnowledgeName string // 知识库名称
	DocumentsId   int64  // 文档ID
}

type IndexAsyncReq struct {
	Docs          []*schema.Document
	KnowledgeName string // 知识库名称
	DocumentsId   int64  // 文档ID
}

type IndexAsyncByDocsIDReq struct {
	DocsIDs       []string
	KnowledgeName string // 知识库名称
	DocumentsId   int64  // 文档ID
	try           int    // es 数据同步会有部分延迟，尝试多次
}

func (x *Rag) Index(ctx context.Context, req *IndexReq) (ids []string, err error) {
	return nil, nil
}
