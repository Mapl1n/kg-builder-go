package router

import (
	"log"

	"kg-builder-go/internal/config"
	"kg-builder-go/internal/handler"
	"kg-builder-go/internal/ner"
	"kg-builder-go/internal/relation"
	"kg-builder-go/internal/store"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
)

func Setup(cfg *config.Config) *gin.Engine {
	es, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{cfg.ESURL()}})
	if err != nil {
		log.Fatalf("ES: %v", err)
	}

	// Services
	recognizer := ner.New()
	extractor := relation.New()
	graphStore := store.NewGraphStore(es, cfg.ESIndex)
	graphStore.Init()

	// Handlers
	builderH := handler.NewBuilderHandler(recognizer, extractor, graphStore, cfg.TikaURL)
	queryH := handler.NewQueryHandler(graphStore)

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" { c.AbortWithStatus(204); return }
		c.Next()
	})

	r.GET("/", serveWebUI)
	r.GET("/api/health", func(c *gin.Context) { c.JSON(200, gin.H{"status":"ok"}) })

	api := r.Group("/api")
	{
		api.POST("/build", builderH.Build)
		api.GET("/search", queryH.Search)
		api.GET("/graph", queryH.Graph)
		api.GET("/entity-types", queryH.EntityTypes)
	}

	return r
}
