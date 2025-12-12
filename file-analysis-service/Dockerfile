# Этап сборки (Builder)
FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main ./cmd

# Финальный этап (Runner)
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .

# Открываем порт сервиса (Analysis Service у нас на 8082)
EXPOSE 8082

CMD ["./main"]
