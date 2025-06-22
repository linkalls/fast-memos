package handlers

import (
	"errors"
	"fmt"
	"github.com/linkalls/fast-memos/database"
	"github.com/linkalls/fast-memos/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CreateMemoInput struct {
	Title   string `json:"title" xml:"title" form:"title" validate:"required"`
	Content string `json:"content" xml:"content" form:"content"`
}

type UpdateMemoInput struct {
	Title   *string `json:"title,omitempty" xml:"title,omitempty" form:"title,omitempty"`
	Content *string `json:"content,omitempty" xml:"content,omitempty" form:"content,omitempty"`
}

// CreateMemo は新しいメモを作成します
func CreateMemo(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User ID not found in context"})
	}

	input := new(CreateMemoInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON", "details": err.Error()})
	}

	if input.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title is required"})
	}

	memo := models.Memo{
		Title:   input.Title,
		Content: input.Content,
		UserID:  userID,
	}

	result := database.DB.Create(&memo)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create memo", "details": result.Error.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(memo)
}

// GetMemos は認証されたユーザーのすべてのメモを取得します
func GetMemos(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User ID not found in context"})
	}

	var memos []models.Memo
	// ユーザーIDでフィルタリングし、作成日時の降順で取得
	result := database.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&memos)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not retrieve memos", "details": result.Error.Error()})
	}

	return c.JSON(memos)
}

// GetMemo は認証されたユーザーの特定のメモを取得します
func GetMemo(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User ID not found in context"})
	}

	memoID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid memo ID"})
	}

	var memo models.Memo
	result := database.DB.Where("id = ? AND user_id = ?", uint(memoID), userID).First(&memo)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Memo not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not retrieve memo", "details": result.Error.Error()})
	}

	return c.JSON(memo)
}

// UpdateMemo は認証されたユーザーの特定のメモを更新します
func UpdateMemo(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User ID not found in context"})
	}

	memoID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid memo ID"})
	}

	input := new(UpdateMemoInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON", "details": err.Error()})
	}

	var memo models.Memo
	result := database.DB.Where("id = ? AND user_id = ?", uint(memoID), userID).First(&memo)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Memo not found or not owned by user"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not retrieve memo for update", "details": result.Error.Error()})
	}

	// 更新するフィールドのみを適用
	if input.Title != nil {
		if *input.Title == "" { // タイトルを空にすることは許可しない場合
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title cannot be empty"})
		}
		memo.Title = *input.Title
	}
	if input.Content != nil {
		memo.Content = *input.Content
	}

	saveResult := database.DB.Save(&memo)
	if saveResult.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update memo", "details": saveResult.Error.Error()})
	}

	return c.JSON(memo)
}

// DeleteMemo は認証されたユーザーの特定のメモを削除します
func DeleteMemo(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User ID not found in context"})
	}

	memoID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid memo ID"})
	}

	// まずメモが存在し、かつユーザーが所有しているか確認
	var memo models.Memo
	result := database.DB.Where("id = ? AND user_id = ?", uint(memoID), userID).First(&memo)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Memo not found or not owned by user"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not retrieve memo for deletion", "details": result.Error.Error()})
	}

	// 削除実行
	deleteResult := database.DB.Delete(&models.Memo{}, uint(memoID)) // GORMは主キーで削除する場合、オブジェクトを渡すか、モデルとIDを指定する
	if deleteResult.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not delete memo", "details": deleteResult.Error.Error()})
	}
	if deleteResult.RowsAffected == 0 {
		// このケースは通常、上記のFirstチェックで捕捉されるはずだが、念のため
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Memo not found or already deleted"})
	}
	
	// 成功メッセージを返す
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": fmt.Sprintf("Memo with ID %d deleted successfully", memoID)})
}

// SearchMemos は認証されたユーザーのメモをキーワードで検索します
func SearchMemos(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User ID not found in context"})
	}

	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Search query 'q' is required"})
	}

	var memos []models.Memo
	// タイトルまたは本文にキーワードを含むメモを検索 (LIKE句、大文字小文字を区別しない)
	// SQLiteでは ILIKE が直接サポートされていない場合があるため、lower関数で対応
	searchTerm := "%" + query + "%"
	result := database.DB.Where("user_id = ? AND (lower(title) LIKE lower(?) OR lower(content) LIKE lower(?))", userID, searchTerm, searchTerm).Order("created_at desc").Find(&memos)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not search memos", "details": result.Error.Error()})
	}

	return c.JSON(memos)
}
