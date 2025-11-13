package rag

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/core/common"
	"github.com/everfid-ever/ThinkForge/internal/logic/chat"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// ChatStream 流式输出接口
func (c *ControllerV1) ChatStream(ctx context.Context, req *v1.ChatStreamReq) (res *v1.ChatStreamRes, err error) {
	var streamReader *schema.StreamReader[*schema.Message]
	// 获取检索结果
	retriever, err := c.Retriever(ctx, &v1.RetrieverReq{
		Question:      req.Question,
		TopK:          req.TopK,
		Score:         req.Score,
		KnowledgeName: req.KnowledgeName,
	})
	if err != nil {
		g.Log().Error(ctx, err)
		return
	}
	// 获取Chat实例
	chatI := chat.GetChat()
	// 获取流式响应
	streamReader, err = chatI.GetAnswerStream(ctx, req.ConvID, retriever.Document, req.Question)
	if err != nil {
		g.Log().Error(ctx, err)
		return &v1.ChatStreamRes{}, nil
	}
	defer streamReader.Close()
	err = common.StreamResponse(ctx, streamReader, retriever.Document)
	if err != nil {
		g.Log().Error(ctx, err)
		return
	}
	return &v1.ChatStreamRes{}, nil
}

// writeSSEData 写入普通数据事件（模型生成的文本）
func writeSSEData(resp *ghttp.Response, data string) {
	if len(data) == 0 {
		return
	}
	g.Log().Infof(context.Background(), "data: %s", data)
	// SSE 格式要求以 "data:" 开头，后跟换行
	resp.Writeln(fmt.Sprintf("data:%s\n", data))
	resp.Flush() // 立即刷新（非常重要，否则数据会被缓冲）
}

func writeSSEDone(resp *ghttp.Response) {
	resp.Writeln(fmt.Sprintf("data:%s\n", "[DONE]"))
	resp.Flush()
}

func writeSSEDocuments(resp *ghttp.Response, data string) {
	resp.Writeln(fmt.Sprintf("documents:%s\n", data))
	resp.Flush()
}

// writeSSEError 写入SSE错误
func writeSSEError(resp *ghttp.Response, err error) {
	g.Log().Error(context.Background(), err)
	resp.Writeln(fmt.Sprintf("event: error\ndata: %s\n\n", err.Error()))
	resp.Flush()
}
