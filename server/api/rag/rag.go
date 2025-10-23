package rag

import (
	"context"

	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
)

type IRetrieverV1 interface {
	Retriever(ctx context.Context, req *v1.RetrieverReq) (res *v1.RetrieverRes, err error)
}
