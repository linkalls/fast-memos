package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/linkalls/fast-memos/auth"
	"github.com/linkalls/fast-memos/database"
	"github.com/linkalls/fast-memos/models"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert" // アサーションライブラリ
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testApp *fiber.App
var testDB *gorm.DB

// setupTestApp はテスト用のFiberアプリとデータベースを初期化します
func setupTestApp() *fiber.App {
	// インメモリSQLiteデータベースの設定
	var err error
	testDB, err = gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{LogLevel: logger.Silent}, // テスト中はサイレントに
		),
	})
	if err != nil {
		log.Fatalf("Failed to connect to in-memory database: %v", err)
	}
	database.DB = testDB // グローバルなDBインスタンスをテスト用DBに置き換え

	// モデルのマイグレーション
	err = testDB.AutoMigrate(&models.User{}, &models.Memo{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	app := fiber.New()

	api := app.Group("/api")
	authRoutes := api.Group("/auth")
	authRoutes.Post("/register", RegisterUser)
	authRoutes.Post("/login", LoginUser)

	// メモ関連のルートもテストで必要ならここに追加
	memoRoutes := api.Group("/memos", auth.AuthMiddleware()) // AuthMiddlewareをグローバルに適用
	memoRoutes.Post("/", CreateMemo)
	memoRoutes.Get("/", GetMemos)
	memoRoutes.Get("/search", SearchMemos) 
	memoRoutes.Get("/:id", GetMemo)
	memoRoutes.Put("/:id", UpdateMemo)
	memoRoutes.Delete("/:id", DeleteMemo)


	return app
}

// TestMain はテストのセットアップとティアダウンを行います
func TestMain(m *testing.M) {
	testApp = setupTestApp()
	code := m.Run() // テストを実行
	// ティアダウン処理 (もしあれば)
	sqlDB, _ := testDB.DB()
	sqlDB.Close()
	os.Exit(code)
}

// clearDatabase はテスト間でデータベースをクリーンアップします
func clearDatabase() {
	testDB.Exec("DELETE FROM memos")
	testDB.Exec("DELETE FROM users")
	// 他のテーブルも必要に応じてクリア
}

func TestRegisterUser(t *testing.T) {
	clearDatabase() // 各テストの前にDBをクリア

	payload := `{"username": "testuser", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(req, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.Equal(t, "testuser", result["username"])
	userID, ok := result["id"].(string)
	assert.True(t, ok, "User ID should be a string")
	assert.NotEmpty(t, userID, "User ID should not be empty")

	// Store the created user ID for later tests if needed, or fetch from DB
}

func TestRegisterUser_DuplicateUsername(t *testing.T) {
	clearDatabase()
	// 最初のユーザー登録
	payload := `{"username": "testuser", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := testApp.Test(req, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// 同じユーザー名で再度登録
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(payload))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := testApp.Test(req2, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	// SQLiteの場合、GORMは `ErrDuplicatedKey` を返さないことがあるので、
	// ハンドラ側でエラーメッセージに基づいてStatusInternalServerErrorを返す想定
	assert.Equal(t, http.StatusInternalServerError, resp2.StatusCode) // または適切なエラーコード
}


func TestLoginUser(t *testing.T) {
	clearDatabase()
	// まずユーザーを登録
	registerPayload := `{"username": "loginuser", "password": "password123"}`
	reqRegister := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(registerPayload))
	reqRegister.Header.Set("Content-Type", "application/json")
	respRegister, _ := testApp.Test(reqRegister, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusCreated, respRegister.StatusCode)

	// DBから登録されたユーザーIDを取得 (テストのため)
	var createdUser models.User
	errUser := testDB.Where("username = ?", "loginuser").First(&createdUser).Error
	assert.NoError(t, errUser)
	assert.NotEmpty(t, createdUser.ID)


	loginPayload := `{"username": "loginuser", "password": "password123"}`
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(reqLogin, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	json.Unmarshal(body, &result)
	assert.NotEmpty(t, result["token"])

	// Verify JWT token content (user_id)
	tokenString := result["token"]
	validatedUserID, errToken := auth.ValidateJWT(tokenString)
	assert.NoError(t, errToken)
	assert.Equal(t, createdUser.ID, validatedUserID)
}

func TestLoginUser_InvalidCredentials(t *testing.T) {
	clearDatabase()
	// ユーザー登録
	registerPayload := `{"username": "loginuser", "password": "password123"}`
	reqRegister := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(registerPayload))
	reqRegister.Header.Set("Content-Type", "application/json")
	testApp.Test(reqRegister, -1) // タイムアウトを無効化

	// 間違ったパスワードでログイン
	loginPayload := `{"username": "loginuser", "password": "wrongpassword"}`
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")

	resp, err := testApp.Test(reqLogin, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
