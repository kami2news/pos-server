package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/kami2news/pos-server/db"
	"github.com/kami2news/pos-server/handlers"
	"github.com/kami2news/pos-server/middleware"
)

func main() {
	// ── DB接続 ──────────────────────────────────────────────
	database, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("DB接続失敗: %v", err)
	}
	defer database.Close()

	// ── マイグレーション ────────────────────────────────────
	if err := db.Migrate(database); err != nil {
		log.Fatalf("マイグレーション失敗: %v", err)
	}
	log.Println("DB migration OK")

	// ── Ginルーター ─────────────────────────────────────────
	r := gin.Default()
	r.Use(corsMiddleware())

	// ヘルスチェック（Railwayのヘルスチェック用）
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	h := handlers.New(database)

	// ── 認証（JWT不要） ──────────────────────────────────────
	r.POST("/auth/register", h.Register) // 管理者のみ（X-Admin-Secret必須）
	r.POST("/auth/login", h.Login)

	// ── 認証必須ルート ──────────────────────────────────────
	auth := r.Group("/", middleware.Auth())
	{
		auth.GET("/auth/me", h.Me)

		api := auth.Group("/api")

		// 営業日
		api.POST("/business-days", h.OpenBusinessDay)
		api.PATCH("/business-days/close", h.CloseBusinessDay)
		api.GET("/business-days/current", h.CurrentBusinessDay)

		// 商品
		api.GET("/products", h.ListProducts)
		api.POST("/products", h.CreateProduct)
		api.PATCH("/products/:id", h.UpdateProduct)
		api.DELETE("/products/:id", h.DeleteProduct)

		// 注文
		api.GET("/orders", h.ListOrders)
		api.POST("/orders", h.CreateOrder)
		api.PATCH("/orders/:id/complete", h.CompleteOrder)
		api.PATCH("/orders/:id/ready", h.ReadyOrder)
		api.PATCH("/orders/:id/handover", h.HandoverOrder)
		api.PATCH("/orders/:id/cancel", h.CancelOrder)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("サーバー起動失敗: %v", err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Admin-Secret")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
