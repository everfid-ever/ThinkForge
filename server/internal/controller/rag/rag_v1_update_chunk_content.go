package rag

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/schema"
	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/core"
	"github.com/everfid-ever/ThinkForge/internal/logic/knowledge"
	"github.com/everfid-ever/ThinkForge/internal/logic/rag"
	"github.com/everfid-ever/ThinkForge/internal/model/entity"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

func (c *ControllerV1) UpdateChunkContent(ctx context.Context, req *v1.UpdateChunkContentReq) (res *v1.UpdateChunkContentRes, err error) {
	chunk, err := knowledge.GetChunkById(ctx, req.Id)
	if err != nil {
		g.Log().Errorf(ctx, "GetChunkById failed, err=%v", err)
		return
	}

	document, err := knowledge.GetDocumentById(ctx, chunk.KnowledgeDocId)
	if err != nil {
		g.Log().Errorf(ctx, "GetDocumentById failed, err=%v", err)
		return
	}

	knowledgeName := document.KnowledgeBaseName

	err = knowledge.UpdateChunkByIds(ctx, []int64{req.Id}, entity.KnowledgeChunks{
		Content: req.Content,
	})
	if err != nil {
		g.Log().Errorf(ctx, "UpdateChunkByIds failed, err=%v", err)
		return
	}

	go func() {
		// 等待一段时间确保数据库更新完成
		time.Sleep(time.Millisecond * 500)

		ctxN := gctx.New()
		defer func() {
			if e := recover(); e != nil {
				g.Log().Errorf(ctxN, "recover updateChunkContent failed, err=%v", e)
			}
		}()

		doc := &schema.Document{
			ID:      chunk.ChunkId,
			Content: req.Content,
		}

		if chunk.Ext != "" {
			extData := map[string]any{}
			if err := sonic.Unmarshal([]byte(chunk.Ext), &extData); err == nil {
				doc.MetaData = extData
			}
		}

		// 调用异步索引更新
		ragSvr := rag.GetRagSvr()
		asyncReq := &core.IndexAsyncReq{
			Docs:          []*schema.Document{doc},
			KnowledgeName: knowledgeName,
			DocumentsId:   chunk.KnowledgeDocId,
		}

		_, err = ragSvr.IndexAsync(ctxN, asyncReq)
		if err != nil {
			g.Log().Errorf(ctxN, "IndexAsync failed, err=%v", err)
		} else {
			g.Log().Infof(ctxN, "Chunk content updated and reindexed successfully, chunk_id=%d", req.Id)
		}
	}()

	return
}
