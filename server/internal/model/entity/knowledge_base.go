// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// KnowledgeBase is the golang structure for table knowledge_base.
type KnowledgeBase struct {
	Id          int64       `json:"id"          orm:"id"          description:"Primary Key ID"`         	   // 主键ID
	Name        string      `json:"name"        orm:"name"        description:"Knowledge base name"`           // 知识库名称
	Description string      `json:"description" orm:"description" description:"Knowledge base description"`    // 知识库描述
	Category    string      `json:"category"    orm:"category"    description:"Knowledge base category"`       // 知识库分类
	Status      int         `json:"status"      orm:"status"      description:"Status: 0-disabled, 1-enabled"` // 状态：0-禁用，1-启用
	CreateTime  *gtime.Time `json:"createTime"  orm:"create_time" description:"Creation time"`        		   // 创建时间
	UpdateTime  *gtime.Time `json:"updateTime"  orm:"update_time" description:"Update time"`  	  		       // 更新时间
}
