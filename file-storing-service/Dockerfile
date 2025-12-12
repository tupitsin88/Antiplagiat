# Этап сборки (Builder)
FROM golang:alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей и скачиваем их
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код и собираем бинарник
COPY . .
RUN go build -o main ./cmd

# Финальный этап (Runner)
FROM alpine:latest

WORKDIR /root/

# Копируем скомпилированный бинарник из этапа сборки
COPY --from=builder /app/main .

# Открываем порт сервиса (Storing Service у нас на 8081)
EXPOSE 8081

# Запускаем
CMD ["./main"]
