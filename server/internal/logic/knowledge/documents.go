package knowledge

import (
	"context"
	"fmt"

	"github.com/everfid-ever/ThinkForge/internal/dao"
	"github.com/everfid-ever/ThinkForge/internal/model/entity"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

const (
	defaultPageSize = 10
	maxPageSize     = 100
)

// SaveDocumentsInfo 保存文档信息
func SaveDocumentsInfo(ctx context.Context, documents entity.KnowledgeDocuments) (id int64, err error) {
	result, err := dao.KnowledgeDocuments.Ctx(ctx).Data(documents).Insert()
	if err != nil {
		g.Log().Errorf(ctx, "failed to save document information: %+v, Error: %v", documents, err)
		return 0, fmt.Errorf("failed to save document information: %w", err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve insert ID: %w", err)
	}

	g.Log().Infof(ctx, "document saved successfully, ID: %d", id)
	return id, nil
}

// UpdateDocumentsStatus 更新文档状态
func UpdateDocumentsStatus(ctx context.Context, documentsId int64, status int) error {
	data := g.Map{
		"status": status,
	}

	_, err := dao.KnowledgeDocuments.Ctx(ctx).Where("id", documentsId).Data(data).Update()
	if err != nil {
		g.Log().Errorf(ctx, "document status update failed: ID=%d, Error: %v", documentsId, err)
	}

	return err
}

// GetDocumentById 根据ID获取文档信息
func GetDocumentById(ctx context.Context, id int64) (document entity.KnowledgeDocuments, err error) {
	g.Log().Debugf(ctx, "get document information: ID=%d", id)

	err = dao.KnowledgeDocuments.Ctx(ctx).Where("id", id).Scan(&document)
	if err != nil {
		g.Log().Errorf(ctx, "failed to retrieve document information: ID=%d, Error: %v", id, err)
		return document, fmt.Errorf("failed to retrieve document information: %w", err)
	}

	return document, nil
}

// GetDocumentsList 获取文档列表
func GetDocumentsList(ctx context.Context, where entity.KnowledgeDocuments, page int, pageSize int) (documents []entity.KnowledgeDocuments, total int, err error) {
	// 参数验证和默认值设置
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	model := dao.KnowledgeDocuments.Ctx(ctx)
	if where.KnowledgeBaseName != "" {
		model = model.Where("knowledge_base_name", where.KnowledgeBaseName)
	}

	total, err = model.Count()
	if err != nil {
		g.Log().Errorf(ctx, "failed to retrieve total number of documents: %v", err)
		return nil, 0, fmt.Errorf("failed to retrieve total number of documents: %w", err)
	}

	if total == 0 {
		return nil, 0, nil
	}

	err = model.Page(page, pageSize).
		Order("created_at desc").
		Scan(&documents)
	if err != nil {
		g.Log().Errorf(ctx, "failed to retrieve document list: %v", err)
		return nil, 0, fmt.Errorf("failed to retrieve document list: %w", err)
	}

	return documents, total, nil
}

// DeleteDocument 删除文档及其相关数据
func DeleteDocument(ctx context.Context, id int64) error {
	g.Log().Debugf(ctx, "felete document: ID=%d", id)

	return dao.KnowledgeDocuments.Ctx(ctx).Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 先删除文档块
		_, err := dao.KnowledgeChunks.Ctx(ctx).TX(tx).Where("knowledge_doc_id", id).Delete()
		if err != nil {
			g.Log().Errorf(ctx, "failed to delete document block: ID=%d, Error: %v", id, err)
			return fmt.Errorf("failed to delete document block: %w", err)
		}

		// 再删除文档
		result, err := dao.KnowledgeDocuments.Ctx(ctx).TX(tx).Where("id", id).Delete()
		if err != nil {
			g.Log().Errorf(ctx, "document deletion failed: ID=%d, Error: %v", id, err)
			return fmt.Errorf("document deletion failed: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to retrieve the number of affected rows: %w", err)
		}
		if affected == 0 {
			return fmt.Errorf("document does not exist")
		}

		g.Log().Infof(ctx, "document deleted successfully: ID=%d", id)
		return nil
	})
}
