# ステージ1: ビルドステージ
FROM golang:1.22-alpine AS builder

# 作業ディレクトリを設定
WORKDIR /app

# Goモジュールの依存関係をコピーしてダウンロード
# go.mod と go.sum のみをコピーして、依存関係のレイヤーをキャッシュする
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# アプリケーションをビルド
# CGO_ENABLED=0 で静的リンクされたバイナリを生成 (alpineで実行するために重要)
# -ldflags="-w -s" でデバッグ情報を削除し、バイナリサイズを削減
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /app/main .

# ステージ2: 実行ステージ
FROM alpine:latest

# ルート証明書をインストール (HTTPS通信などに必要になる場合がある)
RUN apk --no-cache add ca-certificates

# 作業ディレクトリを設定
WORKDIR /app

# ビルドステージから実行可能ファイルをコピー
COPY --from=builder /app/main /app/main

# アプリケーションがリッスンするポートを公開
EXPOSE 3000

# コンテナ起動時のコマンド
# アプリケーションはカレントディレクトリの memo_app.db を使用すると想定
CMD ["/app/main"]
