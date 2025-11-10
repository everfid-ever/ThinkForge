package rag

import (
	"context"

	v1 "github.com/everfid-ever/ThinkForge/api/rag/v1"
	"github.com/everfid-ever/ThinkForge/internal/dao"
	"github.com/everfid-ever/ThinkForge/internal/model/do"
)

// KBCreate 处理“创建知识库”的接口请求。
// 功能：接收客户端传来的知识库信息（名称、描述、分类等），插入数据库并返回新ID。
func (c *ControllerV1) KBCreate(ctx context.Context, req *v1.KBCreateReq) (res *v1.KBCreateRes, err error) {
	// 向 knowledge_base 表中插入一条记录
	insertId, err := dao.KnowledgeBase.Ctx(ctx).Data(do.KnowledgeBase{
		Name:        req.Name,
		Status:      v1.StatusOK, // 默认状态为 OK
		Description: req.Description,
		Category:    req.Category,
	}).InsertAndGetId() // 插入并返回自增主键ID

	if err != nil {
		return nil, err // 插入失败，返回错误
	}

	// 返回响应结果（包含新创建的 ID）
	res = &v1.KBCreateRes{
		Id: insertId,
	}
	return
}

// KBDelete 处理“删除知识库”的请求。
// 功能：根据知识库主键 ID 删除对应记录。
func (c *ControllerV1) KBDelete(ctx context.Context, req *v1.KBDeleteReq) (res *v1.KBDeleteRes, err error) {
	// 根据主键 ID 删除记录
	_, err = dao.KnowledgeBase.Ctx(ctx).WherePri(req.Id).Delete()
	return
}

// KBGetList 查询知识库列表。
// 功能：根据条件（状态、名称、分类）从数据库中获取符合条件的知识库记录。
func (c *ControllerV1) KBGetList(ctx context.Context, req *v1.KBGetListReq) (res *v1.KBGetListRes, err error) {
	res = &v1.KBGetListRes{}
	// 构建筛选条件
	err = dao.KnowledgeBase.Ctx(ctx).Where(do.KnowledgeBase{
		Status:   req.Status,
		Name:     req.Name,
		Category: req.Category,
	}).Scan(&res.List) // 扫描结果到响应结构体
	return
}

// KBGetOne 获取单个知识库详情。
// 功能：根据 ID 查询单条知识库记录。
func (c *ControllerV1) KBGetOne(ctx context.Context, req *v1.KBGetOneReq) (res *v1.KBGetOneRes, err error) {
	res = &v1.KBGetOneRes{}
	// 通过主键 ID 查询单条记录
	err = dao.KnowledgeBase.Ctx(ctx).WherePri(req.Id).Scan(&res.KnowledgeBase)
	return
}

// KBUpdate 更新知识库信息。
// 功能：根据 ID 修改知识库的名称、状态、描述、分类等字段。
func (c *ControllerV1) KBUpdate(ctx context.Context, req *v1.KBUpdateReq) (res *v1.KBUpdateRes, err error) {
	// 按主键更新记录
	_, err = dao.KnowledgeBase.Ctx(ctx).Data(do.KnowledgeBase{
		Name:        req.Name,
		Status:      req.Status,
		Description: req.Description,
		Category:    req.Category,
	}).WherePri(req.Id).Update()
	return
}
