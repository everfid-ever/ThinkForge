package rag

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bytedance/sonic"
	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/core/common"
	"github.com/everfid-ever/ThinkForge/internal/logic/chat"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

// ChatStream 流式输出接口
func (c *ControllerV1) ChatStream(ctx context.Context, req *v1.ChatStreamReq) (res *v1.ChatStreamRes, err error) {
	// 获取HTTP请求/响应对象
	httpReq := ghttp.RequestFromCtx(ctx)
	httpResp := httpReq.Response
	// 设置响应头，让前端以SSE方式接收数据
	httpResp.Header().Set("Content-Type", "text/event-stream") // 表示这是事件流
	httpResp.Header().Set("Cache-Control", "no-cache")         // 禁止缓存
	httpResp.Header().Set("Connection", "keep-alive")          // 保持连接不断开
	httpResp.Header().Set("X-Accel-Buffering", "no")           // 禁用 Nginx 缓冲（重要，否则数据无法实时推送）
	httpResp.Header().Set("Access-Control-Allow-Origin", "*")  // 允许跨域访问

	// 获取检索结果，基于问题检索相关文档（RAG 前半部分：Retrieve）
	retriever, err := c.Retriever(ctx, &v1.RetrieverReq{
		Question:      req.Question,      // 用户问题
		TopK:          req.TopK,          // 取前K个结果
		Score:         req.Score,         // 相似度阈值
		KnowledgeName: req.KnowledgeName, // 知识库名称
	})
	if err != nil {
		writeSSEError(httpResp, err) // 将错误通过 SSE 推送给前端
		return &v1.ChatStreamRes{}, nil
	}
	// 构建初始流数据结构
	sd := &common.StreamData{
		Id:       uuid.NewString(),   // 唯一消息ID
		Created:  time.Now().Unix(),  // 消息创建时间戳
		Document: retriever.Document, // 检索到的参考文档
	}
	marshal, _ := sonic.Marshal(sd)
	writeSSEDocuments(httpResp, string(marshal)) // 推送文档信息给前端（可显示引用来源）
	// 获取Chat实例
	chatI := chat.GetChat()
	// 调用 LLM 的流式回答接口
	// 内部会调用 x.GetAnswerStream() → 启动 goroutine + stream.Copy
	streamReader, err := chatI.GetAnswerStream(ctx, req.ConvID, retriever.Document, req.Question)
	if err != nil {
		writeSSEError(httpResp, err)
		return &v1.ChatStreamRes{}, nil
	}
	defer streamReader.Close()

	// 不断从流中读取生成内容并推送给前端
	for {
		chunk, err := streamReader.Recv() // 每次接收一小段内容
		if err == io.EOF {
			break
		}
		if err != nil {
			writeSSEError(httpResp, err)
			break
		}
		if len(chunk.Content) == 0 {
			continue // 跳过空内容
		}

		sd.Content = chunk.Content
		marshal, _ := sonic.Marshal(sd)
		// 推送这一小段内容到前端（浏览器会实时显示）
		writeSSEData(httpResp, string(marshal))
	}
	// 发送结束事件
	writeSSEDone(httpResp)

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
