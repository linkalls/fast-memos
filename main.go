package main

import (
	"log"
	"github.com/linkalls/fast-memos/auth"
	"github.com/linkalls/fast-memos/database"
	"github.com/linkalls/fast-memos/handlers"

	"github.com/gofiber/contrib/cors" // CORSミドルウェアをインポート
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

	// CORSミドルウェアの設定
	// フロントエンドの開発サーバー (例: Viteのデフォルト localhost:5173) からのアクセスを許可
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173", // フロントエンドのURL
		AllowHeaders:  "Origin, Content-Type, Accept, Authorization", // Authorizationヘッダーも許可
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
	}))

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

	// 静的ファイル配信 (Reactアプリのビルド成果物)
	// この設定はAPIルートより後に記述することが重要
	// "/" のルートに来たリクエストは ./frontend/dist ディレクトリのファイルを探す
	// index.html がルートになり、React Routerなどがクライアントサイドルーティングを処理する
	// Fiber v2.12.0 以降では app.Static に Single: true オプションが利用可能で、
	// SPAのルーティング (存在しないパスへのリクエストをindex.htmlにフォールバックする) に便利です。
	// 今回は、まず基本的な配信設定のみ行います。
	// app.Static("/", "./frontend/dist")
	// より確実なSPA対応のため、ファイルが存在しない場合は index.html を返すようにします。
	app.Static("/", "./frontend/dist", fiber.Static{
		Index:    "index.html",
		Compress: true,
		// Single: true, // もしFiberのバージョンが対応していればこちらがよりシンプル
		NotFound: func(c *fiber.Ctx) error { // Single: true が使えない場合の代替
			return c.SendFile("./frontend/dist/index.html")
		},
	})
	
	// サーバーを指定ポートで起動 (例: 3000)
	// ポートは環境変数などから取得するのが望ましい
	port := "3000" 
	log.Printf("Server is starting on port %s\n", port)
	err := app.Listen(":" + port)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
