package model

import "time"

// Entity 知识图谱实体
type Entity struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`   // person, org, contract, date, money, location
	DocID    string   `json:"doc_id"`
	DocName  string   `json:"doc_name"`
	Mentions []string `json:"mentions,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Relation 实体关系
type Relation struct {
	ID       string `json:"id"`
	Subject  string `json:"subject"`   // entity ID
	Predicate string `json:"predicate"` // relation type
	Object   string `json:"object"`    // entity ID
	DocID    string `json:"doc_id"`
	Evidence string `json:"evidence"`  // 原文证据
}

// GraphData 前端可视化数据
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type GraphNode struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Group int    `json:"group"`
	DocID string `json:"doc_id,omitempty"`
	Size  int    `json:"symbolSize"`
}

type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
}

// EntityTypeConfig 实体类型配置
type EntityTypeConfig struct {
	Type     string   `json:"type"`
	Label    string   `json:"label"`
	Color    string   `json:"color"`
	Keywords []string `json:"keywords"`
}

var EntityTypes = []EntityTypeConfig{
	{Type: "person", Label: "人物", Color: "#3b82f6", Keywords: []string{"法定代表人","负责人","联系人","经理","董事长","总经理","股东"}},
	{Type: "org", Label: "机构", Color: "#22c55e", Keywords: []string{"公司","有限公司","集团","企业","单位","部门","机构"}},
	{Type: "contract", Label: "合同", Color: "#f59e0b", Keywords: []string{"合同","协议","约定","签订","甲方","乙方","条款"}},
	{Type: "date", Label: "日期", Color: "#8b5cf6", Keywords: []string{"日期","时间","期限"}},
	{Type: "money", Label: "金额", Color: "#ef4444", Keywords: []string{"金额","元","万元","美元","费用","价款"}},
	{Type: "location", Label: "地点", Color: "#06b6d4", Keywords: []string{"地址","地点","位于","省","市","区"}},
}
