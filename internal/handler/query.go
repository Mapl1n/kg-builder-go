package handler

import (
	"kg-builder-go/internal/store"

	"github.com/gin-gonic/gin"
)

type QueryHandler struct {
	store *store.GraphStore
}

func NewQueryHandler(store *store.GraphStore) *QueryHandler {
	return &QueryHandler{store: store}
}

// Search 搜索实体
func (h *QueryHandler) Search(c *gin.Context) {
	name := c.Query("name")
	entityType := c.Query("type")
	docID := c.Query("doc_id")

	entities, err := h.store.SearchEntity(name, entityType, docID)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "data": entities})
}

// Graph 获取图谱可视化数据
func (h *QueryHandler) Graph(c *gin.Context) {
	entityID := c.Query("entity_id")
	depth := 2
	if d, ok := c.GetQuery("depth"); ok {
		_ = d
	}

	data, err := h.store.GetGraphData(entityID, depth)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "data": data})
}

// EntityTypes 返回实体类型配置
func (h *QueryHandler) EntityTypes(c *gin.Context) {
	c.JSON(200, gin.H{"code": 0, "data": nil})
}
