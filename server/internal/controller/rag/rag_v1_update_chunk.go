package rag

import (
	"context"

	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/internal/logic/knowledge"
	"github.com/everfid-ever/ThinkForge/internal/model/entity"
)

func (c *ControllerV1) UpdateChunk(ctx context.Context, req *v1.UpdateChunkReq) (res *v1.UpdateChunkRes, err error) {
	err = knowledge.UpdateChunkByIds(ctx, req.Ids, entity.KnowledgeChunks{
		Status: req.Status,
	})
	if err != nil {
		return
	}

	return
}
