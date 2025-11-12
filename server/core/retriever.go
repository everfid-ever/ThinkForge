package core

import (
	"context"
	"github.com/cloudwego/eino/schema"
	"github.com/everfid-ever/ThinkForge/core/retriever"
	"github.com/gogf/gf/v2/frame/g"
	"sync"
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
	var (
		used        = ""
		relatedDocs = &sync.Map{}
		docNum      = 0
	)
	req.rankScore = req.Score
	// 大于1的需要-1
	if req.rankScore >= 1 {
		req.rankScore -= 1
	}
	// 最多尝试N次,后续做成可配置
	for i := 0; i < 3; i++ {
		question := req.Query
		var (
			messages []*schema.Message
			generate *schema.Message
			docs     []*schema.Document
			pass     bool
		)
		messages, err = getOptimizedQueryMessages(used, question, req.KnowledgeName)
		if err != nil {
			return
		}
		generate, err = x.cm.Generate(ctx, messages)
		if err != nil {
			return
		}
		docs, err = retriever.NewRerank(ctx, req.optQuery, docs, req.TopK)
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
		wg := &sync.WaitGroup{}
		for _, doc := range docs {
			// 分数不够的直接不管
			if doc.Score() < req.rankScore {
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				pass, err = x.grader.Related(ctx, doc, req.Query)
				if err != nil {
					return
				}
				if pass {
					req.excludeIDs = append(req.excludeIDs, doc.ID) // 后续不要检索这个_id对应的数据
					relatedDocs.Store(doc.ID, doc)
					docNum++
				} else {
					g.Log().Infof(ctx, "not doc score: %v, related: %v", doc.Score(), doc.Content)
				}
			}()
		}
		wg.Wait()
		// 数量不够就再次检索
		if docNum < req.TopK {
			continue
		}
		// 数量够了，就直接返回
		rDocs := make([]*schema.Document, 0, req.TopK)
		relatedDocs.Range(func(key, value any) bool {
			rDocs = append(rDocs, value.(*schema.Document))
			return true
		})
		pass, err = x.grader.Retriever(ctx, rDocs, req.Query)
		if err != nil {
			return
		}
		if pass {
			return rDocs, nil
		}
	}
	return
}
