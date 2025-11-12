package core

import (
	"context"
	"github.com/cloudwego/eino/schema"
	"github.com/everfid-ever/ThinkForge/core/common"
	"github.com/everfid-ever/ThinkForge/core/retriever"
	"github.com/gogf/gf/v2/frame/g"
	"sort"
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
	)
	req.rankScore = req.Score
	// 大于1的需要-1
	if req.rankScore >= 1 {
		req.rankScore -= 1
	}
	wgg := &sync.WaitGroup{}
	rewriteModel, err := common.GetRewriteModel(ctx, nil)
	if err != nil {
		return
	}
	// 最多尝试N次,后续做成可配置
	for i := 0; i < 3; i++ {
		wgg.Add(1)
		question := req.Query
		var (
			messages []*schema.Message
			rewrite  *schema.Message
		)
		messages, err = getOptimizedQueryMessages(used, question, req.KnowledgeName)
		if err != nil {
			return
		}
		rewrite, err = rewriteModel.Generate(ctx, messages)
		if err != nil {
			return
		}
		docs, err = retriever.NewRerank(ctx, req.optQuery, docs, req.TopK)
		if err != nil {
			return
		}
		optimizedQuery := rewrite.Content
		used += optimizedQuery + " "
		req.optQuery = optimizedQuery
		onceDo := func() {
			var (
				docs   []*schema.Document
				qaDocs []*schema.Document
			)
			docs, err = x.retrieve(ctx, req, false)
			if err != nil {
				g.Log().Errorf(ctx, "retrieve failed, err=%v", err)
				return
			}
			qaDocs, err = x.retrieve(ctx, req, true)
			if err != nil {
				g.Log().Errorf(ctx, "qa retrieve failed, err=%v", err)
				return
			}
			docs = append(docs, qaDocs...)
			// 去重
			dm := make(map[string]*schema.Document)
			for _, doc := range docs {
				dm[doc.ID] = doc
				continue
			}
			docs = make([]*schema.Document, 0, len(dm))
			for _, doc := range dm {
				docs = append(docs, doc)
			}
			docs, err = retriever.NewRerank(ctx, req.optQuery, docs, req.TopK)
			if err != nil {
				g.Log().Errorf(ctx, "Rerank failed, err=%v", err)
				return
			}
			// wg := &sync.WaitGroup{}
			// for _, doc := range docs {
			// 	// 分数不够的直接不管
			// 	if doc.Score() < req.rankScore {
			// 		g.Log().Infof(ctx, "not rankScore score: %v, related: %v", doc.Score(), doc.Content)
			// 		continue
			// 	}
			// 	wg.Add(1)
			// 	go func() {
			// 		defer wg.Done()
			// 		// 检查下检索到的结果是否和用户问题相关
			// 		// 代价太大，没意义
			// 		pass, err = x.grader.Related(ctx, doc, req.Query)
			// 		if err != nil {
			// 			return
			// 		}
			// 		if pass {
			// 			req.excludeIDs = append(req.excludeIDs, doc.ID) // 后续不要检索这个_id对应的数据
			// 			relatedDocs.Store(doc.ID, doc)
			// 			docNum++
			// 		} else {
			// 			g.Log().Infof(ctx, "not doc score: %v, related: %v", doc.Score(), doc.Content)
			// 		}
			// 	}()
			// }
			// wg.Wait()
			for _, doc := range docs {
				if doc.Score() < req.rankScore {
					g.Log().Debugf(ctx, "score less: %v, related: %v", doc.Score(), doc.Content)
					continue
				}
				if old, e := relatedDocs.LoadOrStore(doc.ID, doc); e {
					// 保存较高分的结果
					if doc.Score() > old.(*schema.Document).Score() {
						relatedDocs.Store(doc.ID, doc)
					}
				}
			}
		}
		// 数量够了，就直接返回
		// rDocs := make([]*schema.Document, 0, req.TopK)
		// relatedDocs.Range(func(key, value any) bool {
		// 	rDocs = append(rDocs, value.(*schema.Document))
		// 	return true
		// })
		// pass, err = x.grader.Retriever(ctx, rDocs, req.Query)
		// if err != nil {
		// 	return
		// }
		// if pass {
		// 	break
		// }
	}
	wgg.Wait()
	// 最后数量不够，就返回所有数据
	relatedDocs.Range(func(key, value any) bool {
		msg = append(msg, value.(*schema.Document))
		return true
	})
	sort.Slice(msg, func(i, j int) bool {
		return msg[i].Score() > msg[j].Score()
	})
	if len(msg) > req.TopK {
		msg = msg[:req.TopK]
	}
	return
}

func (x *Rag) retrieveOnce(ctx context.Context, req *RetrieveReq, qa bool) (msg []*schema.Document, err error) {
	return
}
