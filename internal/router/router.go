package router

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"time"

	"kg-builder-go/internal/config"
	"kg-builder-go/internal/ner"
	"kg-builder-go/internal/relation"
	"kg-builder-go/internal/store"
	"kg-builder-go/pkg/docparser"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func Setup(cfg *config.Config) *gin.Engine {
	// Use local graph store (zero external dependencies)
	recognizer := ner.New()
	extractor := relation.New()
	graphStore := store.NewLocalGraphStore()

	log.Println("[KG] running in standalone mode (in-memory graph store + local NER)")

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" { c.AbortWithStatus(204); return }
		c.Next()
	})

	r.GET("/", serveWebUI)
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "mode": "standalone"})
	})

	api := r.Group("/api")
	{
		api.POST("/build", func(c *gin.Context) {
			file, header, err := c.Request.FormFile("file")
			if err != nil {
				c.JSON(400, gin.H{"code": 400, "message": "请选择文件"})
				return
			}
			defer file.Close()
			rawData, _ := io.ReadAll(file)

			// 本地 docparser 优先（纯 Go PDF/DOCX/TXT）
			text, err := docparser.Parse(rawData, header.Filename)
			if err != nil {
				// 降级到 Tika
				client := &http.Client{Timeout: 60 * time.Second}
				req, _ := http.NewRequest("PUT", cfg.TikaURL+"/tika", bytes.NewReader(rawData))
				req.Header.Set("Accept", "text/plain")
				if resp, tErr := client.Do(req); tErr == nil {
					defer resp.Body.Close()
					content, _ := io.ReadAll(resp.Body)
					text = string(content)
				}
			}
			if text == "" {
				c.JSON(503, gin.H{"code": 503, "message": "无法解析该文档，请上传 PDF/DOCX/TXT 文件"})
				return
			}

			docID := uuid.New().String()
			entities := recognizer.Extract(docID, header.Filename, text)
			relations := extractor.Extract(docID, entities, text)

			now := time.Now()
			for _, e := range entities {
				e.CreatedAt = now
				graphStore.SaveEntity(e)
			}
			for _, r := range relations {
				graphStore.SaveRelation(r)
			}

			maxShow := 20
			showE := entities
			showR := relations
			if len(entities) > maxShow { showE = entities[:maxShow] }
			if len(relations) > maxShow { showR = relations[:maxShow] }

			c.JSON(200, gin.H{
				"code": 0,
				"data": gin.H{
					"doc_id": docID, "filename": header.Filename,
					"entity_count": len(entities), "relation_count": len(relations),
					"text_len": len(text), "entities": showE, "relations": showR,
				},
			})
		})

		api.GET("/search", func(c *gin.Context) {
			entities := graphStore.SearchEntity(c.Query("name"), c.Query("type"), c.Query("doc_id"))
			c.JSON(200, gin.H{"code": 0, "data": entities})
		})

		api.GET("/graph", func(c *gin.Context) {
			depth := 2
			data, err := graphStore.GetGraphData(c.Query("entity_id"), depth)
			if err != nil { c.JSON(500, gin.H{"code": 500}); return }
			c.JSON(200, gin.H{"code": 0, "data": data})
		})

		api.GET("/entity-types", func(c *gin.Context) {
			c.JSON(200, gin.H{"code": 0, "data": nil})
		})
	}

	return r
}
