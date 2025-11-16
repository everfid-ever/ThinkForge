package rag

import (
	"context"

	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/internal/logic/knowledge"
	"github.com/everfid-ever/ThinkForge/internal/logic/rag"
	"github.com/gogf/gf/v2/frame/g"
)

func (c *ControllerV1) ChunkDelete(ctx context.Context, req *v1.ChunkDeleteReq) (res *v1.ChunkDeleteRes, err error) {
	svr := rag.GetRagSvr()

	chunk, err := knowledge.GetChunkById(ctx, req.Id)
	if err != nil {
		g.Log().Errorf(ctx, "DeleteDocumentAndChunks: GetChunkById failed for id %v, err: %v", req.Id, err)
		return
	}

	err = svr.DeleteDocument(ctx, chunk.ChunkId)
	if err != nil {
		g.Log().Errorf(ctx, "DeleteDocumentAndChunks: ES DeleteByQuery failed for docId %v, err: %v", chunk.ChunkId, err)
		return
	}

	err = knowledge.DeleteChunkById(ctx, req.Id)
	return
}
