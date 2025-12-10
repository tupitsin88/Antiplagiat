# syntax=docker/dockerfile:1
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd

FROM scratch
COPY --from=builder /app/app /app
EXPOSE 8082
ENTRYPOINT ["/app"]
