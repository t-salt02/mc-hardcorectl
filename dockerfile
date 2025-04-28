FROM golang:1.24.1 as builder
WORKDIR /src

# 1) go.mod / go.sum だけコピーして依存キャッシュを作る
COPY go.mod go.sum ./
RUN go mod download

# 2) 残りのソースをコピー
COPY . .

# 3) cmd/bot をビルド   ←★ターゲットを明示！
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /bot/bot ./cmd/bot

# ---------- runtime stage ----------
FROM gcr.io/distroless/static
COPY --from=builder /bot/bot /bot
ENTRYPOINT ["/bot"]
