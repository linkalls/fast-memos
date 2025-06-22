package handlers

import (
	"errors"
	"fmt"
	"github.com/linkalls/fast-memos/database"
	"github.com/linkalls/fast-memos/models"
	"github.com/linkalls/fast-memos/utils" // 追加
	"strings"                             // 追加
	// "strconv" // 不要になるのでコメントアウトまたは削除

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Helper function to convert []string to comma-separated string
func relatedIDsToString(ids []string) string {
	if ids == nil {
		return ""
	}
	return strings.Join(ids, ",")
}

// Helper function to convert comma-separated string to []string
func stringToRelatedIDs(s string) []string {
	if s == "" {
		return []string{} // 空の文字列の場合は空のスライスを返す
	}
	ids := strings.Split(s, ",")
	var result []string
	for _, id := range ids {
		trimmedID := strings.TrimSpace(id)
		if trimmedID != "" {
			result = append(result, trimmedID)
		}
	}
	// Splitが空文字列に対して [""] を返すことがあるため、resultが空でもnilではない場合がある
	if len(result) == 0 && len(ids) == 1 && ids[0] == "" {
		return []string{}
	}
	if result == nil { // もし上の条件で漏れた場合（例：idsが元々nilだった場合など、Joinの結果が空になるケース）
		return []string{}
	}
	return result
}


type CreateMemoInput struct {
	Title          string   `json:"title" xml:"title" form:"title" validate:"required"`
	Content        string   `json:"content" xml:"content" form:"content"`
	RelatedMemoIDs []string `json:"related_memo_ids" xml:"related_memo_ids" form:"related_memo_ids"`
}

type UpdateMemoInput struct {
	Title          *string   `json:"title,omitempty" xml:"title,omitempty" form:"title,omitempty"`
	Content        *string   `json:"content,omitempty" xml:"content,omitempty" form:"content,omitempty"`
	RelatedMemoIDs *[]string `json:"related_memo_ids,omitempty" xml:"related_memo_ids,omitempty" form:"related_memo_ids,omitempty"` // ポインタ型に変更, omitempty を推奨
}

// CreateMemo は新しいメモを作成します
func CreateMemo(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string) // stringに変更
	if !ok || userID == "" { // userIDが空の場合もエラー
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found or invalid in context"})
	}

	input := new(CreateMemoInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON", "details": err.Error()})
	}

	if input.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title is required"})
	}

	memoID := utils.GenerateID() // 新しいメモIDを生成

	memo := models.Memo{
		ID:                  memoID, // 設定
		Title:               input.Title,
		Content:             input.Content,
		UserID:              userID, // string型
		RelatedMemoIDsStore: relatedIDsToString(input.RelatedMemoIDs), // 変換して保存
	}

	result := database.DB.Create(&memo)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create memo", "details": result.Error.Error()})
	}

	// レスポンスのために RelatedMemoIDs をセット
	memo.RelatedMemoIDs = stringToRelatedIDs(memo.RelatedMemoIDsStore)

	return c.Status(fiber.StatusCreated).JSON(memo)
}

// GetMemos は認証されたユーザーのすべてのメモを取得します
func GetMemos(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string) // stringに変更
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found or invalid in context"})
	}

	var memos []models.Memo
	// ユーザーIDでフィルタリングし、作成日時の降順で取得
	result := database.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&memos)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not retrieve memos", "details": result.Error.Error()})
	}

	// 各メモについて RelatedMemoIDs を設定
	for i := range memos {
		memos[i].RelatedMemoIDs = stringToRelatedIDs(memos[i].RelatedMemoIDsStore)
	}

	return c.JSON(memos)
}

// GetMemo は認証されたユーザーの特定のメモを取得します
func GetMemo(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string) // stringに変更
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found or invalid in context"})
	}

	memoID := c.Params("id") // string ID
	if memoID == "" { // パスパラメータが空かどうかのチェック
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Memo ID is required"})
	}

	var memo models.Memo
	result := database.DB.Where("id = ? AND user_id = ?", memoID, userID).First(&memo)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Memo not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not retrieve memo", "details": result.Error.Error()})
	}

	// RelatedMemoIDs を設定
	memo.RelatedMemoIDs = stringToRelatedIDs(memo.RelatedMemoIDsStore)

	return c.JSON(memo)
}

// UpdateMemo は認証されたユーザーの特定のメモを更新します
func UpdateMemo(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string) // stringに変更
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found or invalid in context"})
	}

	memoID := c.Params("id") // string ID
	if memoID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Memo ID is required"})
	}

	input := new(UpdateMemoInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON", "details": err.Error()})
	}

	var memo models.Memo
	// まずはIDだけで取得（UserIDによる絞り込みは所有権確認のため）
	result := database.DB.Where("id = ? AND user_id = ?", memoID, userID).First(&memo)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Memo not found or not owned by user"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not retrieve memo for update", "details": result.Error.Error()})
	}

	// 更新フラグ
	updated := false

	// 更新するフィールドのみを適用
	if input.Title != nil {
		if *input.Title == "" { // タイトルを空にすることは許可しない場合
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Title cannot be empty"})
		}
		if memo.Title != *input.Title {
			memo.Title = *input.Title
			updated = true
		}
	}
	if input.Content != nil {
		if memo.Content != *input.Content {
			memo.Content = *input.Content
			updated = true
		}
	}

	// RelatedMemoIDsの更新処理
	if input.RelatedMemoIDs != nil { // ポインタがnilでなければ、キーが存在し、値がnullでないことを意味する
		newRelatedStore := relatedIDsToString(*input.RelatedMemoIDs) // ポインタをデリファレンス
		if memo.RelatedMemoIDsStore != newRelatedStore {
			memo.RelatedMemoIDsStore = newRelatedStore
			updated = true
		}
	}
	// input.RelatedMemoIDs が nil の場合はキーが存在しないか値がnullだったので、何もしない (既存の値を維持)

	if !updated {
		 // 何も更新がない場合 (input.RelatedMemoIDsがnilで、他のフィールドも更新なしの場合)
         memo.RelatedMemoIDs = stringToRelatedIDs(memo.RelatedMemoIDsStore)
         return c.JSON(memo)
    }

	saveResult := database.DB.Save(&memo)
	if saveResult.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update memo", "details": saveResult.Error.Error()})
	}

	// レスポンスのために RelatedMemoIDs をセット
	memo.RelatedMemoIDs = stringToRelatedIDs(memo.RelatedMemoIDsStore)

	return c.JSON(memo)
}

// DeleteMemo は認証されたユーザーの特定のメモを削除します
func DeleteMemo(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string) // stringに変更
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found or invalid in context"})
	}

	memoID := c.Params("id") // string ID
	if memoID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Memo ID is required"})
	}

	// まずメモが存在し、かつユーザーが所有しているか確認
	var memo models.Memo // この変数は削除確認には使うが、直接削除には使わない
	result := database.DB.Where("id = ? AND user_id = ?", memoID, userID).First(&memo)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Memo not found or not owned by user"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not retrieve memo for deletion", "details": result.Error.Error()})
	}

	// 削除実行 (文字列IDの場合は明示的にWHERE句を指定する方が安全)
	deleteResult := database.DB.Where("id = ?", memoID).Delete(&models.Memo{})
	if deleteResult.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not delete memo", "details": deleteResult.Error.Error()})
	}
	if deleteResult.RowsAffected == 0 {
		// このケースは通常、上記のFirstチェックで捕捉されるはずだが、念のため
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Memo not found or already deleted (during delete operation)"})
	}
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": fmt.Sprintf("Memo with ID %s deleted successfully", memoID)})
}

// SearchMemos は認証されたユーザーのメモをキーワードで検索します
func SearchMemos(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string) // stringに変更
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID not found or invalid in context"})
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

	// 各メモについて RelatedMemoIDs を設定
	for i := range memos {
		memos[i].RelatedMemoIDs = stringToRelatedIDs(memos[i].RelatedMemoIDsStore)
	}

	return c.JSON(memos)
}
