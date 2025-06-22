# ステージ1: Goビルドステージ (バックエンド)
FROM golang:1.24-alpine AS go-builder

WORKDIR /app

# 必要なビルドツールとsqlite開発ヘッダをインストール
RUN apk add --no-cache build-base sqlite-dev

# Goモジュールの依存関係をコピーしてダウンロード
COPY go.mod go.sum ./
RUN go mod tidy

# ソースコードをコピー
COPY . .

# アプリケーションをビルド
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o /app/main .

# ステージ2: 最終実行ステージ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Goビルドステージから実行可能ファイルをコピー
COPY --from=go-builder /app/main /app/main

# 必要に応じて public ディレクトリ等の静的ファイルをコピー
COPY public ./public

EXPOSE 3000

# コンテナ起動時のコマンド
CMD ["/app/main"]
