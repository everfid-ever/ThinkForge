package indexer

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"sync"

	"github.com/cloudwego/eino/schema"
	"github.com/everfid-ever/ThinkForge/core/common"
)

func qa(ctx context.Context, docs []*schema.Document) (output []*schema.Document, err error) {
	var knowledgeName string
	if value, ok := ctx.Value(common.KnowledgeName).(string); ok {
		knowledgeName = value
	} else {
		err = fmt.Errorf("必须提供知识库名称")
		return
	}
	wg := &sync.WaitGroup{}
	for _, doc := range docs {
		wg.Add(1)
		go func(doc *schema.Document) {
			defer wg.Done()
			qaContent, e := getQAContent(ctx, doc, knowledgeName)
			if e != nil {
				g.Log().Errorf(ctx, "getQAContent failed, err=%v", e)
				return
			}
			// 生成QA和内容放在一个chunk的不同字段
			doc.MetaData[common.FieldQAContent] = qaContent
		}(doc)
	}
	wg.Wait()
	return docs, nil
}

func getQAContent(ctx context.Context, doc *schema.Document, knowledgeName string) (qaContent string, err error) {
	// 已经有数据了就不要再生成了
	if s, ok := doc.MetaData[common.FieldQAContent].(string); ok && len(s) > 0 {
		return s, nil
	}
	cm, err := common.GetQAModel(ctx, nil)
	if err != nil {
		return
	}
	generate, err := cm.Generate(ctx, []*schema.Message{
		{
			Role: schema.System,
			Content: fmt.Sprintf("You are a professional question generation assistant whose task is to extract or generate possible questions from given text. You don't need to answer these questions, just generate the questions themselves.\n"+
				"The knowledge base name is: %s\n\n"+
				"Output format: \n"+
				"- Each question should be on a separate line\n"+
				"- Questions must end with a question mark\n"+
				"- Avoiding problems of repetition or semantic similarity\n\n"+
				"Generation rules: \n"+
				"- The generated questions must be strictly based on the text content and cannot be fictionalized outside of the text.\n"+
				"- Prioritize generating factual questions (such as who, when, where, and how).\n"+
				"- For complex texts, multi-level questions (basic facts + reasoning questions) can be generated.\n"+
				"- Subjective or open-ended questions (such as \"What do you think...?\") are prohibited."+
				"- The quantity should be controlled at 3-5.", knowledgeName),
		},
		{
			Role:    schema.User,
			Content: doc.Content,
		},
	})
	if err != nil {
		return
	}
	qaContent = generate.Content
	return
}
