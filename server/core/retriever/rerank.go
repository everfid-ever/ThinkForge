package retriever

import (
	"context"
	
	"github.com/cloudwego/eino/schema"
	"github.com/everfid-ever/ThinkForge/core/rerank"
)

func NewRerank(ctx context.Context, query string, docs []*schema.Document, topK int) (output []*schema.Document, err error) {
	output, err = rerank.Rerank(ctx, query, docs, topK)
	if err != nil {
		return
	}
	return
}
