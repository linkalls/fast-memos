package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/linkalls/fast-memos/auth"
	"github.com/linkalls/fast-memos/database"
	"github.com/linkalls/fast-memos/models"
	"github.com/linkalls/fast-memos/utils"
)

// WebLoginUser - Web UI用のログインハンドラー
func WebLoginUser(c *fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "" || password == "" {
		return c.Render("login", fiber.Map{
			"Title": "Login",
			"Error": "ユーザー名とパスワードを入力してください",
		})
	}

	// 通常のAPIログイン処理を呼び出し
	user := models.User{Username: username, Password: password}

	var existingUser models.User
	if err := database.DB.Where("username = ?", user.Username).First(&existingUser).Error; err != nil {
		return c.Render("login", fiber.Map{
			"Title": "Login",
			"Error": "ユーザーが見つかりません",
		})
	}

	if !auth.CheckPasswordHash(user.Password, existingUser.Password) {
		return c.Render("login", fiber.Map{
			"Title": "Login",
			"Error": "パスワードが正しくありません",
		})
	}

	// ログイン成功時はuser_idをCookieにセット
	c.Cookie(&fiber.Cookie{
		Name:     "user_id",
		Value:    existingUser.ID,
		Path:     "/",
		HTTPOnly: true,
		Secure:   false, // 本番はtrue
	})
	return c.Redirect("/")
}

// WebRegisterUser - Web UI用の登録ハンドラー
func WebRegisterUser(c *fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "" || password == "" {
		return c.Render("register", fiber.Map{
			"Title": "Register",
			"Error": "ユーザー名とパスワードを入力してください",
		})
	}

	// ユーザーの重複チェック
	var existingUser models.User
	if err := database.DB.Where("username = ?", username).First(&existingUser).Error; err == nil {
		return c.Render("register", fiber.Map{
			"Title": "Register",
			"Error": "このユーザー名は既に使用されています",
		})
	}

	// パスワードをハッシュ化
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return c.Render("register", fiber.Map{
			"Title": "Register",
			"Error": "パスワードの処理に失敗しました",
		})
	}

	// 新しいユーザーを作成
	user := models.User{
		ID:       utils.GenerateID(),
		Username: username,
		Password: hashedPassword,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return c.Render("register", fiber.Map{
			"Title": "Register",
			"Error": "ユーザーの作成に失敗しました",
		})
	}

	// 登録成功時はログインページにリダイレクト
	return c.Redirect("/login")
}

// WebCreateMemo - Web UI用のメモ作成ハンドラー
func WebCreateMemo(c *fiber.Ctx) error {
	content := c.FormValue("content")
	category := c.FormValue("category")

	if content == "" {
		return c.Redirect("/?error=content_required")
	}

	userID := c.Cookies("user_id")
	if userID == "" {
		return c.Redirect("/login")
	}

	memo := models.Memo{
		ID:       utils.GenerateID(),
		Title:    "", // タイトルは空でOK
		Content:  content,
		Category: category,
		UserID:   userID,
	}

	if err := database.DB.Create(&memo).Error; err != nil {
		return c.Redirect("/?error=failed_to_create_memo")
	}

	accept := c.Get("Accept")
	if accept == "text/vnd.turbo-stream.html" {
		return c.Render("memo.turbo-stream", memo, "text/vnd.turbo-stream.html")
	}

	return c.Redirect("/")
}

// WebDeleteMemo - Web UI用のメモ削除ハンドラー
func WebDeleteMemo(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Redirect("/")
	}
	database.DB.Delete(&models.Memo{}, "id = ?", id)
	// Turbo Stream対応
	accept := c.Get("Accept")
	if accept == "text/vnd.turbo-stream.html" {
		// 削除用Turbo Stream返却
		return c.SendString(`<turbo-stream action="remove" target="memo-` + id + `"></turbo-stream>`)
	}
	return c.Redirect("/")
}

// WebEditMemo - 編集フォーム表示
func WebEditMemo(c *fiber.Ctx) error {
	id := c.Params("id")
	var memo models.Memo
	if err := database.DB.First(&memo, "id = ?", id).Error; err != nil {
		return c.Redirect("/")
	}
	return c.Render("edit_memo", fiber.Map{
		"Memo": memo,
	})
}

// WebUpdateMemo - 編集内容を保存
func WebUpdateMemo(c *fiber.Ctx) error {
	id := c.Params("id")
	title := c.FormValue("title")
	content := c.FormValue("content")
	category := c.FormValue("category")
	if id == "" || title == "" || content == "" {
		return c.Redirect("/")
	}
	database.DB.Model(&models.Memo{}).Where("id = ?", id).Updates(map[string]interface{}{
		"title":    title,
		"content":  content,
		"category": category,
	})
	return c.Redirect("/")
}
