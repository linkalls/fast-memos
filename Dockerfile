# ステージ1: フロントエンドビルドステージ
FROM node:20-alpine AS frontend-builder

# frontendディレクトリを作業ディレクトリに設定
WORKDIR /app/frontend

# package.json と package-lock.json* (npm) または yarn.lock (yarn) をコピー
# package-lock.json が存在すれば npm ci を使うためにワイルドカードでコピー
COPY frontend/package.json frontend/package-lock.json* ./

# npm ci は package-lock.json に基づいてクリーンインストールを行うため推奨
# package-lock.json がない場合は npm install を実行
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

# frontendのソースコードをコピー
COPY frontend/ ./

# フロントエンドをビルド
# VITE_API_BASE_URL は、Goサーバーから配信する場合、通常は設定不要か、
# フロントエンド側でAPIリクエストを相対パス (`/api/...`) にすることで対応します。
# 必要であれば ARG VITE_API_BASE_URL で定義し、`docker build --build-arg` で渡すことも可能です。
RUN npm run build
# ビルド成果物は /app/frontend/dist に作成される

# ステージ2: Goビルドステージ (バックエンド)
FROM golang:1.24-alpine AS go-builder

WORKDIR /app

# Goモジュールの依存関係をコピーしてダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
# .dockerignore で frontend/node_modules などが除外されていることを前提とします。
COPY . .

# アプリケーションをビルド
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /app/main .

# ステージ3: 最終実行ステージ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Goビルドステージから実行可能ファイルをコピー
COPY --from=go-builder /app/main /app/main

# フロントエンドビルドステージからビルドされた静的ファイルをコピー
# main.go の app.Static("/", "./frontend/dist") に対応するよう配置
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

EXPOSE 3000

# コンテナ起動時のコマンド
CMD ["/app/main"]
