package handler

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"kg-builder-go/internal/model"
	"kg-builder-go/internal/ner"
	"kg-builder-go/internal/relation"
	"kg-builder-go/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BuilderHandler struct {
	ner     *ner.Recognizer
	rel     *relation.Extractor
	store   *store.GraphStore
	tikaURL string
}

func NewBuilderHandler(ner *ner.Recognizer, rel *relation.Extractor, store *store.GraphStore, tikaURL string) *BuilderHandler {
	return &BuilderHandler{ner: ner, rel: rel, store: store, tikaURL: tikaURL}
}

// Build 上传文档 → Tika解析 → NER提取 → 关系抽取 → 存储图谱
func (h *BuilderHandler) Build(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请选择文件"})
		return
	}
	defer file.Close()

	rawData, _ := io.ReadAll(file)
	var text string

	// 尝试 Tika 解析，失败则直接当作文本
	client := &http.Client{Timeout: 60 * time.Second}
	req, _ := http.NewRequest("PUT", h.tikaURL+"/tika", bytes.NewReader(rawData))
	req.Header.Set("Accept", "text/plain")
	if resp, err := client.Do(req); err == nil {
		defer resp.Body.Close()
		if content, err := io.ReadAll(resp.Body); err == nil && len(content) > 0 {
			text = string(content)
		}
	}
	if text == "" {
		// Tika 不可用时直接用原始文本
		text = string(rawData)
	}

	docID := uuid.New().String()

	// NER 提取
	entities := h.ner.Extract(docID, header.Filename, text)

	// 关系抽取
	relations := h.rel.Extract(docID, entities, text)

	// 存储到 ES 图
	now := time.Now()
	for _, e := range entities {
		e.CreatedAt = now
		h.store.SaveEntity(e)
	}
	for _, r := range relations {
		h.store.SaveRelation(r)
	}

	// 限制返回数量
	maxShow := 20
	showEntities := entities
	showRelations := relations
	if len(entities) > maxShow {
		showEntities = entities[:maxShow]
	}
	if len(relations) > maxShow {
		showRelations = relations[:maxShow]
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": gin.H{
			"doc_id":         docID,
			"filename":       header.Filename,
			"entity_count":   len(entities),
			"relation_count": len(relations),
			"text_len":       len(text),
			"entities":       showEntities,
			"relations":      showRelations,
		},
	})
	_ = model.EntityTypeConfig{}
}
