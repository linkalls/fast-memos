package main

import (
	"html/template"
	"log"

	"github.com/linkalls/fast-memos/auth"
	"github.com/linkalls/fast-memos/database"
	"github.com/linkalls/fast-memos/handlers"
	"github.com/linkalls/fast-memos/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors" // CORSミドルウェアをインポート
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"github.com/russross/blackfriday/v2"
)

func main() {
	// データベースに接続
	database.ConnectDatabase()

	// HTMLテンプレートエンジンを設定
	engine := html.New("./templates", ".html")
	engine.AddFunc("markdown", func(text string) template.HTML {
		// MarkdownをHTMLに変換
		output := blackfriday.Run([]byte(text))
		return template.HTML(output)
	})

	// Fiberアプリのインスタンスを作成
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// ミドルウェアの設定
	app.Use(logger.New())  // リクエストロガー
	app.Use(recover.New()) // パニックリカバリー

	// CORSミドルウェアの設定
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",                                           // 全てのオリジンを許可
		AllowHeaders: "Origin, Content-Type, Accept, Authorization", // Authorizationヘッダーも許可
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

	// 静的ファイル配信 (publicディレクトリ)
	app.Static("/public", "./public")

	// Web UIルート
	app.Get("/", func(c *fiber.Ctx) error {
		userID := c.Cookies("user_id")
		if userID == "" {
			authHeader := c.Get("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token := authHeader[7:]
				parsedID, err := auth.ParseJWT(token)
				if err == nil {
					userID = parsedID
				}
			}
		}
		if userID == "" {
			return c.Redirect("/login")
		}
		q := c.Query("q")
		var memos []models.Memo
		db := database.DB.Where("user_id = ?", userID)
		if q != "" {
			like := "%" + q + "%"
			db = db.Where("title LIKE ? OR content LIKE ? OR category LIKE ?", like, like, like)
		}
		db.Order("created_at desc").Find(&memos)
		return c.Render("index", fiber.Map{
			"Title": "Fast Memos",
			"Memos": memos,
			"Query": q,
		})
	})

	app.Get("/login", func(c *fiber.Ctx) error {
		return c.Render("login", fiber.Map{
			"Title": "Login",
		})
	})

	app.Get("/register", func(c *fiber.Ctx) error {
		return c.Render("register", fiber.Map{
			"Title": "Register",
		})
	})

	// フォーム送信用のPOSTルート
	app.Post("/login", handlers.WebLoginUser)
	app.Post("/register", handlers.WebRegisterUser)
	app.Post("/memos", handlers.WebCreateMemo)
	app.Post("/memos/:id/delete", handlers.WebDeleteMemo)
	app.Get("/memos/:id/edit", handlers.WebEditMemo)
	app.Post("/memos/:id/edit", handlers.WebUpdateMemo)

	// サーバーを指定ポートで起動 (例: 3000)
	// ポートは環境変数などから取得するのが望ましい
	port := "3000"
	log.Printf("Server is starting on port %s\n", port)
	err := app.Listen(":" + port)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
