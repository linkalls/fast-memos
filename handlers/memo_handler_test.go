package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linkalls/fast-memos/models" // testAppのセットアップはauth_handler_test.goのTestMainで行われる想定

	"github.com/stretchr/testify/assert"
)

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

	payload := `{"title": "Test Memo", "content": "This is a test memo."}`
	req := httptest.NewRequest(http.MethodPost, "/api/memos/", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := testApp.Test(req, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var memo models.Memo
	json.Unmarshal(body, &memo)
	assert.Equal(t, "Test Memo", memo.Title)
	assert.Equal(t, "This is a test memo.", memo.Content)
	assert.NotZero(t, memo.ID)
	assert.NotZero(t, memo.UserID)
}

func TestGetMemos(t *testing.T) {
	token := loginTestUser(t, "memouser2", "password123")

	// 最初にメモをいくつか作成
	for i := 0; i < 3; i++ {
		payload := fmt.Sprintf(`{"title": "Memo %d", "content": "Content %d"}`, i+1, i+1)
		reqCreate := httptest.NewRequest(http.MethodPost, "/api/memos/", bytes.NewBufferString(payload))
		reqCreate.Header.Set("Content-Type", "application/json")
		reqCreate.Header.Set("Authorization", "Bearer "+token)
		respCreate, _ := testApp.Test(reqCreate, -1) // タイムアウトを無効化
		assert.Equal(t, http.StatusCreated, respCreate.StatusCode, "Failed to create memo for GetMemos test")
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/api/memos/", nil)
	reqGet.Header.Set("Authorization", "Bearer "+token)

	respGet, err := testApp.Test(reqGet, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, respGet.StatusCode)

	body, _ := io.ReadAll(respGet.Body)
	var memos []models.Memo
	json.Unmarshal(body, &memos)
	assert.Len(t, memos, 3) // 3つのメモが作成されているはず
	assert.Equal(t, "Memo 3", memos[0].Title) // Order("created_at desc") のため新しいものが先頭
}

func TestGetMemo_NotFound(t *testing.T) {
	token := loginTestUser(t, "memouser3", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/memos/99999", nil) // 存在しないID
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := testApp.Test(req, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}


func TestUpdateMemo(t *testing.T) {
	token := loginTestUser(t, "updateuser", "password123")

	// 1. メモを作成
	createPayload := `{"title": "Original Title", "content": "Original Content"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/memos/", bytes.NewBufferString(createPayload))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Authorization", "Bearer "+token)
	respCreate, _ := testApp.Test(reqCreate, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusCreated, respCreate.StatusCode)
	bodyCreate, _ := io.ReadAll(respCreate.Body)
	var createdMemo models.Memo
	json.Unmarshal(bodyCreate, &createdMemo)
	memoID := createdMemo.ID

	// 2. メモを更新
	updatePayload := `{"title": "Updated Title", "content": "Updated Content"}`
	reqUpdate := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/memos/%d", memoID), bytes.NewBufferString(updatePayload))
	reqUpdate.Header.Set("Content-Type", "application/json")
	reqUpdate.Header.Set("Authorization", "Bearer "+token)
	respUpdate, err := testApp.Test(reqUpdate, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, respUpdate.StatusCode)

	bodyUpdate, _ := io.ReadAll(respUpdate.Body)
	var updatedMemo models.Memo
	json.Unmarshal(bodyUpdate, &updatedMemo)
	assert.Equal(t, "Updated Title", updatedMemo.Title)
	assert.Equal(t, "Updated Content", updatedMemo.Content)
}

func TestDeleteMemo(t *testing.T) {
	token := loginTestUser(t, "deleteuser", "password123")

	// 1. メモを作成
	createPayload := `{"title": "To Be Deleted", "content": "Delete me"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/memos/", bytes.NewBufferString(createPayload))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Authorization", "Bearer "+token)
	respCreate, _ := testApp.Test(reqCreate, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusCreated, respCreate.StatusCode)
	bodyCreate, _ := io.ReadAll(respCreate.Body)
	var createdMemo models.Memo
	json.Unmarshal(bodyCreate, &createdMemo)
	memoID := createdMemo.ID

	// 2. メモを削除
	reqDelete := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/memos/%d", memoID), nil)
	reqDelete.Header.Set("Authorization", "Bearer "+token)
	respDelete, err := testApp.Test(reqDelete, -1) // タイムアウトを無効化
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, respDelete.StatusCode)
	
	bodyDelete, _ := io.ReadAll(respDelete.Body)
	var deleteResp map[string]string
	json.Unmarshal(bodyDelete, &deleteResp)
	assert.Contains(t, deleteResp["message"], "deleted successfully")


	// 3. 削除されたメモを取得しようとして404になることを確認
	reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/memos/%d", memoID), nil)
	reqGet.Header.Set("Authorization", "Bearer "+token)
	respGet, _ := testApp.Test(reqGet, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusNotFound, respGet.StatusCode)
}

func TestSearchMemos(t *testing.T) {
	token := loginTestUser(t, "searchuser", "password123")

	// テストデータ作成
	memosToCreate := []models.Memo{
		{Title: "First Test Memo", Content: "Content with keyword Alpha"},
		{Title: "Second Alpha Memo", Content: "Some other text"},
		{Title: "Third Memo", Content: "Another one with Bravo"},
		{Title: "Unique Content", Content: "This is a test for Charlie"},
	}

	for _, memo := range memosToCreate {
		payload := fmt.Sprintf(`{"title": "%s", "content": "%s"}`, memo.Title, memo.Content)
		reqCreate := httptest.NewRequest(http.MethodPost, "/api/memos/", bytes.NewBufferString(payload))
		reqCreate.Header.Set("Content-Type", "application/json")
		reqCreate.Header.Set("Authorization", "Bearer "+token)
		resp, err := testApp.Test(reqCreate, -1) // タイムアウトを無効化
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	// "Alpha" で検索 (タイトルとコンテント)
	reqSearchAlpha := httptest.NewRequest(http.MethodGet, "/api/memos/search?q=Alpha", nil)
	reqSearchAlpha.Header.Set("Authorization", "Bearer "+token)
	respSearchAlpha, _ := testApp.Test(reqSearchAlpha, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusOK, respSearchAlpha.StatusCode)
	bodySearchAlpha, _ := io.ReadAll(respSearchAlpha.Body)
	var resultsAlpha []models.Memo
	json.Unmarshal(bodySearchAlpha, &resultsAlpha)
	assert.Len(t, resultsAlpha, 2) // "First Test Memo" と "Second Alpha Memo"

	// "Charlie" で検索 (コンテントのみ)
	reqSearchCharlie := httptest.NewRequest(http.MethodGet, "/api/memos/search?q=Charlie", nil)
	reqSearchCharlie.Header.Set("Authorization", "Bearer "+token)
	respSearchCharlie, _ := testApp.Test(reqSearchCharlie, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusOK, respSearchCharlie.StatusCode)
	bodySearchCharlie, _ := io.ReadAll(respSearchCharlie.Body)
	var resultsCharlie []models.Memo
	json.Unmarshal(bodySearchCharlie, &resultsCharlie)
	assert.Len(t, resultsCharlie, 1)
	assert.Equal(t, "Unique Content", resultsCharlie[0].Title)
	
	// 存在しないキーワードで検索
	reqSearchNonExistent := httptest.NewRequest(http.MethodGet, "/api/memos/search?q=NonExistentKeyword", nil)
	reqSearchNonExistent.Header.Set("Authorization", "Bearer "+token)
	respSearchNonExistent, _ := testApp.Test(reqSearchNonExistent, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusOK, respSearchNonExistent.StatusCode)
	bodySearchNonExistent, _ := io.ReadAll(respSearchNonExistent.Body)
	var resultsNonExistent []models.Memo
	json.Unmarshal(bodySearchNonExistent, &resultsNonExistent)
	assert.Len(t, resultsNonExistent, 0)

	// クエリなし
	reqSearchNoQuery := httptest.NewRequest(http.MethodGet, "/api/memos/search", nil)
	reqSearchNoQuery.Header.Set("Authorization", "Bearer "+token)
	respSearchNoQuery, _ := testApp.Test(reqSearchNoQuery, -1) // タイムアウトを無効化
	assert.Equal(t, http.StatusBadRequest, respSearchNoQuery.StatusCode)
}
