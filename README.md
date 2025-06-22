# Go Fiber Memo API

これは Go と Fiber フレームワークで作成されたシンプルなメモアプリケーションのバックエンドAPIです。
JWTによる認証機能と、メモのCRUD操作、基本的な全文検索機能を提供します。

## 特徴

-   ユーザー登録とログイン (JWT認証)
-   メモの作成、読み取り、更新、削除 (CRUD)
-   メモのタイトルと内容に対する部分一致検索

## 必要条件

-   Go 1.18 以上

## セットアップと実行

1.  リポジトリをクローンします:
    ```bash
    git clone <repository-url>
    cd memo-app
    ```

2.  依存関係をインストールします:
    ```bash
    go mod tidy
    ```

3.  アプリケーションを実行します:
    ```bash
    go run main.go
    ```
    デフォルトではサーバーは `http://localhost:3000` で起動します。
    データベースファイル `memo_app.db` がプロジェクトルートに作成されます。

## APIエンドポイント

ベースURL: `http://localhost:3000/api`

### 認証 (`/auth`)

-   `POST /auth/register`: 新規ユーザー登録
    -   リクエストボディ: `{"username": "user", "password": "password"}`
    -   成功レスポンス (201): `{"id": 1, "username": "user"}`
-   `POST /auth/login`: ログイン
    -   リクエストボディ: `{"username": "user", "password": "password"}`
    -   成功レスポンス (200): `{"token": "jwt_token_string"}`

### メモ (`/memos`)

**注意:** これらのエンドポイントは認証が必要です。リクエストヘッダーに `Authorization: Bearer <jwt_token>` を含めてください。

-   `POST /memos/`: 新しいメモを作成
    -   リクエストボディ: `{"title": "My Memo", "content": "This is the content."}`
    -   成功レスポンス (201): 作成されたメモオブジェクト
-   `GET /memos/`: 認証ユーザーのすべてのメモを取得
    -   成功レスポンス (200): メモオブジェクトの配列
-   `GET /memos/search?q=<keyword>`: メモを検索
    -   成功レスポンス (200): 条件に一致するメモオブジェクトの配列
-   `GET /memos/:id`: 特定のメモを取得
    -   成功レスポンス (200): メモオブジェクト
    -   失敗レスポンス (404): メモが見つからない場合
-   `PUT /memos/:id`: 特定のメモを更新
    -   リクエストボディ: `{"title": "Updated Title", "content": "Updated content."}` (一部のみでも可)
    -   成功レスポンス (200): 更新されたメモオブジェクト
-   `DELETE /memos/:id`: 特定のメモを削除
    -   成功レスポンス (200): `{"message": "Memo with ID X deleted successfully"}`

## テスト

プロジェクトのルートディレクトリで以下のコマンドを実行します:
```bash
go test ./... -v
```

## 今後の改善点 (TODO)

-   より詳細な入力バリデーションの追加
-   設定ファイル (`.env` や `config.json`) の導入
-   より高度な全文検索機能 (外部検索エンジンの利用など)
-   Docker化
-   Swagger/OpenAPIドキュメントの自動生成
