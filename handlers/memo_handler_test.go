package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	// "strings" // 使用されていないためコメントアウト
	"testing"

	"github.com/linkalls/fast-memos/models" // testAppのセットアップはauth_handler_test.goのTestMainで行われる想定

	"github.com/stretchr/testify/assert"
)

// Helper to read response body for better error messages
func readResponseBody(resp *http.Response) string { // 名前を readResponseBody に変更
	if resp == nil || resp.Body == nil {
		return "<empty response>"
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("<error reading body: %v>", err)
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore body for further reads
	return string(bodyBytes)
}


// loginTestUser はテストユーザーでログインし、JWTトークンを取得します。
// この関数は auth_handler_test.go の clearDatabase と testApp が初期化されている前提です。
func loginTestUser(t *testing.T, username, password string) string {
	clearDatabase() // 既存のユーザーをクリア

	// ユーザー登録
	registerPayload := fmt.Sprintf(`{"username": "%s", "password": "%s"}`, username, password)
	reqRegister := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(registerPayload))
	reqRegister.Header.Set("Content-Type", "application/json")
	respRegister, err := testApp.Test(reqRegister, -1) // -1 はタイムアウトなし
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, respRegister.StatusCode)

	// ログイン
	loginPayload := fmt.Sprintf(`{"username": "%s", "password": "%s"}`, username, password)
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")
	respLogin, err := testApp.Test(reqLogin, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, respLogin.StatusCode)

	body, _ := io.ReadAll(respLogin.Body)
	var result map[string]string
	json.Unmarshal(body, &result)
	token := result["token"]
	assert.NotEmpty(t, token)
	return token
}

func TestCreateMemo(t *testing.T) {
	token := loginTestUser(t, "memouser", "password123")

	// Get user ID from DB for assertion
	var user models.User
	errUser := testDB.Where("username = ?", "memouser").First(&user).Error
	assert.NoError(t, errUser)

	payload := `{"title": "Test Memo", "content": "This is a test memo.", "related_memo_ids": ["id1", "id2"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/memos/", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := testApp.Test(req, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, readResponseBody(resp))

	body, _ := io.ReadAll(resp.Body)
	var memo models.Memo
	json.Unmarshal(body, &memo)
	assert.Equal(t, "Test Memo", memo.Title)
	assert.Equal(t, "This is a test memo.", memo.Content)
	assert.NotEmpty(t, memo.ID, "Memo ID should be a non-empty string")
	assert.IsType(t, "", memo.ID)
	assert.Equal(t, user.ID, memo.UserID)
	assert.ElementsMatch(t, []string{"id1", "id2"}, memo.RelatedMemoIDs)

	// Check DB for RelatedMemoIDsStore
	var dbMemo models.Memo
	testDB.First(&dbMemo, "id = ?", memo.ID)
	assert.Equal(t, "id1,id2", dbMemo.RelatedMemoIDsStore)
}

func TestGetMemos(t *testing.T) {
	token := loginTestUser(t, "memouser2", "password123")

	// 最初にメモをいくつか作成
	for i := 0; i < 3; i++ {
		payload := fmt.Sprintf(`{"title": "Memo %d", "content": "Content %d", "related_memo_ids": ["test%d"]}`, i+1, i+1, i+1)
		reqCreate := httptest.NewRequest(http.MethodPost, "/api/memos/", bytes.NewBufferString(payload))
		reqCreate.Header.Set("Content-Type", "application/json")
		reqCreate.Header.Set("Authorization", "Bearer "+token)
		respCreate, _ := testApp.Test(reqCreate, -1) // タイムアウトを無効化
		assert.Equal(t, http.StatusCreated, respCreate.StatusCode, "Failed to create memo for GetMemos test: "+readResponseBody(respCreate))
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/api/memos/", nil)
	reqGet.Header.Set("Authorization", "Bearer "+token)

	respGet, err := testApp.Test(reqGet, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, respGet.StatusCode, readResponseBody(respGet))

	body, _ := io.ReadAll(respGet.Body)
	var memos []models.Memo
	json.Unmarshal(body, &memos)
	assert.Len(t, memos, 3) // 3つのメモが作成されているはず
	for _, m := range memos {
		assert.NotEmpty(t, m.ID)
		assert.IsType(t, "", m.ID)
		assert.NotEmpty(t, m.UserID) // UserIDも文字列のはず
		assert.IsType(t, "", m.UserID)
		assert.NotNil(t, m.RelatedMemoIDs) // 空スライスかもしれないがnilではない
	}
	assert.Equal(t, "Memo 3", memos[0].Title) // Order("created_at desc") のため新しいものが先頭
	assert.Contains(t, memos[0].RelatedMemoIDs, "test3")
}

func TestGetMemo_NotFound(t *testing.T) {
	token := loginTestUser(t, "memouser3", "password123")
	nonExistentID := "non-existent-uuid-string"
	req := httptest.NewRequest(http.MethodGet, "/api/memos/"+nonExistentID, nil) // 存在しないID
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := testApp.Test(req, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, readResponseBody(resp))
}


func TestUpdateMemo(t *testing.T) {
	token := loginTestUser(t, "updateuser", "password123")

	// 1. メモを作成
	createPayload := `{"title": "Original Title", "content": "Original Content", "related_memo_ids": ["rel1", "rel2"]}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/memos/", bytes.NewBufferString(createPayload))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Authorization", "Bearer "+token)
	respCreate, _ := testApp.Test(reqCreate, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusCreated, respCreate.StatusCode, readResponseBody(respCreate))
	bodyCreate, _ := io.ReadAll(respCreate.Body)
	var createdMemo models.Memo
	json.Unmarshal(bodyCreate, &createdMemo)
	memoID := createdMemo.ID // string ID
	assert.NotEmpty(t, memoID)
	assert.ElementsMatch(t, []string{"rel1", "rel2"}, createdMemo.RelatedMemoIDs)

	// 2. メモを更新 (タイトル、コンテント、関連IDを更新)
	updatePayload := `{"title": "Updated Title", "content": "Updated Content", "related_memo_ids": ["rel3", "rel4"]}`
	reqUpdate := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/memos/%s", memoID), bytes.NewBufferString(updatePayload))
	reqUpdate.Header.Set("Content-Type", "application/json")
	reqUpdate.Header.Set("Authorization", "Bearer "+token)
	respUpdate, err := testApp.Test(reqUpdate, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, respUpdate.StatusCode, readResponseBody(respUpdate))

	bodyUpdate, _ := io.ReadAll(respUpdate.Body)
	var updatedMemo models.Memo
	json.Unmarshal(bodyUpdate, &updatedMemo)
	assert.Equal(t, "Updated Title", updatedMemo.Title)
	assert.Equal(t, "Updated Content", updatedMemo.Content)
	assert.ElementsMatch(t, []string{"rel3", "rel4"}, updatedMemo.RelatedMemoIDs)

	// DBでも確認
	var dbMemo models.Memo
	testDB.First(&dbMemo, "id = ?", memoID)
	assert.Equal(t, "rel3,rel4", dbMemo.RelatedMemoIDsStore)

	// 3. 関連IDのみを空配列に更新
	updatePayloadOnlyRelated := `{"related_memo_ids": []}`
	reqUpdateRelated := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/memos/%s", memoID), bytes.NewBufferString(updatePayloadOnlyRelated))
	reqUpdateRelated.Header.Set("Content-Type", "application/json")
	reqUpdateRelated.Header.Set("Authorization", "Bearer "+token)
	respUpdateRelated, _ := testApp.Test(reqUpdateRelated, -1)
	assert.Equal(t, http.StatusOK, respUpdateRelated.StatusCode, readResponseBody(respUpdateRelated))
	bodyUpdateRelated, _ := io.ReadAll(respUpdateRelated.Body)
	json.Unmarshal(bodyUpdateRelated, &updatedMemo)
	assert.Equal(t, "Updated Title", updatedMemo.Title) // 他は変更されていないはず
	assert.Empty(t, updatedMemo.RelatedMemoIDs)

	// DBでも確認
	testDB.First(&dbMemo, "id = ?", memoID)
	assert.Equal(t, "", dbMemo.RelatedMemoIDsStore)

	// 4. related_memo_ids キーなしで更新 (既存の関連IDは変更されないはず)
	//    ハンドラのロジックでは、キーが存在しない場合 BodyArgs().Has() が false になり、
	//    かつ input.RelatedMemoIDs も nil になるため、この場合は related_memo_ids は更新されない。
	updatePayloadNoRelatedKey := `{"title": "Title Changed Again"}`
	reqUpdateNoRelated := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/memos/%s", memoID), bytes.NewBufferString(updatePayloadNoRelatedKey))
	reqUpdateNoRelated.Header.Set("Content-Type", "application/json")
	reqUpdateNoRelated.Header.Set("Authorization", "Bearer "+token)
	respUpdateNoRelated, _ := testApp.Test(reqUpdateNoRelated, -1)
	assert.Equal(t, http.StatusOK, respUpdateNoRelated.StatusCode, readResponseBody(respUpdateNoRelated))
	bodyUpdateNoRelated, _ := io.ReadAll(respUpdateNoRelated.Body)
	json.Unmarshal(bodyUpdateNoRelated, &updatedMemo)
	assert.Equal(t, "Title Changed Again", updatedMemo.Title)
	assert.Empty(t, updatedMemo.RelatedMemoIDs) // 前のステップでクリアされているため

	testDB.First(&dbMemo, "id = ?", memoID)
	assert.Equal(t, "", dbMemo.RelatedMemoIDsStore) // 変更なし
}

func TestDeleteMemo(t *testing.T) {
	token := loginTestUser(t, "deleteuser", "password123")

	// 1. メモを作成
	createPayload := `{"title": "To Be Deleted", "content": "Delete me", "related_memo_ids": []}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/memos/", bytes.NewBufferString(createPayload))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Authorization", "Bearer "+token)
	respCreate, _ := testApp.Test(reqCreate, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusCreated, respCreate.StatusCode, readResponseBody(respCreate))
	bodyCreate, _ := io.ReadAll(respCreate.Body)
	var createdMemo models.Memo
	json.Unmarshal(bodyCreate, &createdMemo)
	memoID := createdMemo.ID // string ID
	assert.NotEmpty(t, memoID)

	// 2. メモを削除
	reqDelete := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/memos/%s", memoID), nil)
	reqDelete.Header.Set("Authorization", "Bearer "+token)
	respDelete, err := testApp.Test(reqDelete, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, respDelete.StatusCode, readResponseBody(respDelete))
	
	bodyDelete, _ := io.ReadAll(respDelete.Body)
	var deleteResp map[string]string
	json.Unmarshal(bodyDelete, &deleteResp)
	assert.Contains(t, deleteResp["message"], "deleted successfully")


	// 3. 削除されたメモを取得しようとして404になることを確認
	reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/memos/%s", memoID), nil)
	reqGet.Header.Set("Authorization", "Bearer "+token)
	respGet, _ := testApp.Test(reqGet, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusNotFound, respGet.StatusCode, readResponseBody(respGet))
}

func TestSearchMemos(t *testing.T) {
	token := loginTestUser(t, "searchuser", "password123")

	// テストデータ作成
	memosToCreate := []struct {
		Title   string
		Content string
		RelatedMemoIDs []string
	}{
		{Title: "First Test Memo", Content: "Content with keyword Alpha", RelatedMemoIDs: []string{"relA1", "relA2"}},
		{Title: "Second Alpha Memo", Content: "Some other text", RelatedMemoIDs: []string{"relB1"}},
		{Title: "Third Memo", Content: "Another one with Bravo", RelatedMemoIDs: []string{}},
		{Title: "Unique Content", Content: "This is a test for Charlie", RelatedMemoIDs: []string{"relC1"}},
	}

	for _, memoData := range memosToCreate {
		payloadBytes, _ := json.Marshal(map[string]interface{}{"title": memoData.Title, "content": memoData.Content, "related_memo_ids": memoData.RelatedMemoIDs})
		payload := string(payloadBytes)

		reqCreate := httptest.NewRequest(http.MethodPost, "/api/memos/", bytes.NewBufferString(payload))
		reqCreate.Header.Set("Content-Type", "application/json")
		reqCreate.Header.Set("Authorization", "Bearer "+token)
		resp, err := testApp.Test(reqCreate, -1) // タイムアウトを無効化
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode, readResponseBody(resp))
	}

	// "Alpha" で検索 (タイトルとコンテント)
	reqSearchAlpha := httptest.NewRequest(http.MethodGet, "/api/memos/search?q=Alpha", nil)
	reqSearchAlpha.Header.Set("Authorization", "Bearer "+token)
	respSearchAlpha, _ := testApp.Test(reqSearchAlpha, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusOK, respSearchAlpha.StatusCode, readResponseBody(respSearchAlpha))
	bodySearchAlpha, _ := io.ReadAll(respSearchAlpha.Body)
	var resultsAlpha []models.Memo
	json.Unmarshal(bodySearchAlpha, &resultsAlpha)
	assert.Len(t, resultsAlpha, 2) // "First Test Memo" と "Second Alpha Memo"
	// Check related IDs for one of them
	for _, memo := range resultsAlpha {
		if memo.Title == "First Test Memo" {
			assert.ElementsMatch(t, []string{"relA1", "relA2"}, memo.RelatedMemoIDs)
		}
	}


	// "Charlie" で検索 (コンテントのみ)
	reqSearchCharlie := httptest.NewRequest(http.MethodGet, "/api/memos/search?q=Charlie", nil)
	reqSearchCharlie.Header.Set("Authorization", "Bearer "+token)
	respSearchCharlie, _ := testApp.Test(reqSearchCharlie, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusOK, respSearchCharlie.StatusCode, readResponseBody(respSearchCharlie))
	bodySearchCharlie, _ := io.ReadAll(respSearchCharlie.Body)
	var resultsCharlie []models.Memo
	json.Unmarshal(bodySearchCharlie, &resultsCharlie)
	assert.Len(t, resultsCharlie, 1)
	assert.Equal(t, "Unique Content", resultsCharlie[0].Title)
	assert.ElementsMatch(t, []string{"relC1"}, resultsCharlie[0].RelatedMemoIDs)
	
	// 存在しないキーワードで検索
	reqSearchNonExistent := httptest.NewRequest(http.MethodGet, "/api/memos/search?q=NonExistentKeyword", nil)
	reqSearchNonExistent.Header.Set("Authorization", "Bearer "+token)
	respSearchNonExistent, _ := testApp.Test(reqSearchNonExistent, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusOK, respSearchNonExistent.StatusCode, readResponseBody(respSearchNonExistent))
	bodySearchNonExistent, _ := io.ReadAll(respSearchNonExistent.Body)
	var resultsNonExistent []models.Memo
	json.Unmarshal(bodySearchNonExistent, &resultsNonExistent)
	assert.Len(t, resultsNonExistent, 0)

	// クエリなし
	reqSearchNoQuery := httptest.NewRequest(http.MethodGet, "/api/memos/search", nil)
	reqSearchNoQuery.Header.Set("Authorization", "Bearer "+token)
	respSearchNoQuery, _ := testApp.Test(reqSearchNoQuery, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusBadRequest, respSearchNoQuery.StatusCode, readResponseBody(respSearchNoQuery))
}
