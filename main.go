package main

import (
	"log"
	"github.com/linkalls/fast-memos/auth"
	"github.com/linkalls/fast-memos/database"
	"github.com/linkalls/fast-memos/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// データベースに接続
	database.ConnectDatabase()

	// Fiberアプリのインスタンスを作成
	app := fiber.New()

	// ミドルウェアの設定
	app.Use(logger.New())   // リクエストロガー
	app.Use(recover.New()) // パニックリカバリー

	// ルートのグループ化
	api := app.Group("/api")

	// 認証関連のルート
	authRoutes := api.Group("/auth")
	authRoutes.Post("/register", handlers.RegisterUser)
	authRoutes.Post("/login", handlers.LoginUser)

	// メモ関連のルート (認証が必要)
	memoRoutes := api.Group("/memos", auth.AuthMiddleware()) // AuthMiddlewareを適用
	memoRoutes.Post("/", handlers.CreateMemo)
	memoRoutes.Get("/", handlers.GetMemos)
	memoRoutes.Get("/search", handlers.SearchMemos) // 検索エンドポイント
	memoRoutes.Get("/:id", handlers.GetMemo)
	memoRoutes.Put("/:id", handlers.UpdateMemo)
	memoRoutes.Delete("/:id", handlers.DeleteMemo)
	
	// サーバーを指定ポートで起動 (例: 3000)
	// ポートは環境変数などから取得するのが望ましい
	port := "3000" 
	log.Printf("Server is starting on port %s\n", port)
	err := app.Listen(":" + port)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
