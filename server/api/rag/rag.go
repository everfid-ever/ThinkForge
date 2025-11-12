package rag

import (
	"context"

	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
)

// IRagV1 定义了 RAG（Retrieval-Augmented Generation）模块的接口规范。
type IRagV1 interface {
	Chat(ctx context.Context, req *v1.ChatReq) (res *v1.ChatRes, err error)
	Retriever(ctx context.Context, req *v1.RetrieverReq) (res *v1.RetrieverRes, err error)
	Indexer(ctx context.Context, req *v1.IndexerReq) (res *v1.IndexerRes, err error)
	ChatStream(ctx context.Context, req *v1.ChatStreamReq) (res *v1.ChatStreamRes, err error)
	KBCreate(ctx context.Context, req *v1.KBCreateReq) (res *v1.KBCreateRes, err error)
	KBUpdate(ctx context.Context, req *v1.KBUpdateReq) (res *v1.KBUpdateRes, err error)
	KBDelete(ctx context.Context, req *v1.KBDeleteReq) (res *v1.KBDeleteRes, err error)
	KBGetOne(ctx context.Context, req *v1.KBGetOneReq) (res *v1.KBGetOneRes, err error)
	KBGetList(ctx context.Context, req *v1.KBGetListReq) (res *v1.KBGetListRes, err error)
	RetrieverDify(ctx context.Context, req *v1.RetrieverDifyReq) (res *v1.RetrieverDifyRes, err error)
}
