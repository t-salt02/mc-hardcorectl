FROM golang:1.22 as builder
WORKDIR /bot
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bot .

FROM gcr.io/distroless/static
COPY --from=builder /bot/bot /bot
ENTRYPOINT ["/bot"]
