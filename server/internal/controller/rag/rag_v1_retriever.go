package rag

import (
	"context"

	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/internal/logic/rag"
)

// Retriever 处理文档检索请求（/v1/retriever）。
// 主要职责：
//  1. 调用底层 rag 逻辑模块（ragSvr）执行向量检索；
//  2. 过滤或清理无用的元数据字段；
//  3. 将检索结果封装成统一响应结构返回。
//
// 整体流程：
//
//	HTTP 请求 → ControllerV1.Retriever() → ragSvr.Retrieve() → 返回文档结果
func (c *ControllerV1) Retriever(ctx context.Context, req *v1.RetrieverReq) (res *v1.RetrieverRes, err error) {
	// Step 1: 获取 RAG 服务实例。
	// rag.GetRagSvr() 通常返回一个全局或单例的 RAG 服务对象，
	// 内部封装了检索逻辑（例如向量数据库查询、分词匹配、语义相似度计算等）。
	ragSvr := rag.GetRagSvr()

	// Step 2: 校正得分阈值（Score）。
	// 这里逻辑为：若 Score 小于 1，则自动加 1。
	// 推测是为了防止 Score 过低（例如 0.2）时影响检索结果，
	// 或兼容内部评分机制（如内部模型使用 1.x 作为最小阈值）。
	if req.Score < 1.0 {
		req.Score += 1
	}

	// Step 3: 调用 RAG 服务执行检索。
	// 参数说明：
	//   - req.Question: 用户输入的问题；
	//   - req.Score: 文档相似度评分阈值；
	//   - req.TopK: 需要返回的文档数量。
	// 返回值：
	//   - msg: []*schema.Document 格式的文档结果。
	msg, err := ragSvr.Retrieve(req.Question, req.Score, req.TopK)
	if err != nil {
		// 如果检索过程出错（例如向量数据库连接失败），直接返回错误。
		return
	}

	// Step 4: 清理每个文档的 MetaData 字段中不必要的内容。
	// 比如 "_dense_vector" 是内部使用的向量字段，在返回给前端时应删除。
	for _, document := range msg {
		if document.MetaData != nil {
			delete(document.MetaData, "_dense_vector")

			// （可选逻辑）
			// 若需要对得分进行调整，可启用以下代码：
			// if v, e := document.MetaData["_score"]; e {
			//     vf := v.(float64)
			//     document.MetaData["_score"] = vf - 1
			// }
		}
	}

	// Step 5: 构造响应对象。
	res = &v1.RetrieverRes{
		Document: msg, // 返回经过处理的文档列表
	}
	return
}
