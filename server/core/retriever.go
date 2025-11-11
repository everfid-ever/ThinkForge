package core

import (
	"context"
	"github.com/cloudwego/eino/schema"
)

type RetrieveReq struct {
	Query         string   // 检索关键词
	TopK          int      // 检索结果数量
	Score         float64  // 分数阀值(0-2, 0 完全相反，1 毫不相干，2 完全相同,一般需要传入一个大于1的数字，如1.5)
	KnowledgeName string   // 知识库名字
	optQuery      string   // 优化后的检索关键词
	excludeIDs    []string // 要排除的 _id 列表
	rankScore     float64  // 排名分数，原本的score是0-2（实际是1-2），需要在这里改成0-1
}

func (x *RetrieveReq) copy() *RetrieveReq {
	return &RetrieveReq{
		Query:         x.Query,
		TopK:          x.TopK,
		Score:         x.Score,
		KnowledgeName: x.KnowledgeName,
		optQuery:      x.optQuery,
		excludeIDs:    x.excludeIDs,
		rankScore:     x.rankScore,
	}
}

// Retrieve 检索
func (x *Rag) Retrieve(ctx context.Context, req *RetrieveReq) (msg []*schema.Document, err error) {
	used := ""
	for i := 0; i < 5; i++ {
		question := req.Query
		var (
			messages []*schema.Message
			generate *schema.Message
			docs     []*schema.Document
			pass     bool
		)
		messages, err = getMessages(used, question)
		if err != nil {
			return
		}
		generate, err = x.cm.Generate(ctx, messages)
		if err != nil {
			return
		}
		optimizedQuery := generate.Content
		used += optimizedQuery + " "
		req.optQuery = optimizedQuery
		docs, err = x.retrieve(ctx, req)
		if err != nil {
			return
		}
		pass, err = x.grader.Retriever(ctx, docs, req.Query)
		if err != nil {
			return
		}
		if pass {
			return docs, nil
		}
	}
	return
	}
}
