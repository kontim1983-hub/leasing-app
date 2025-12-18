FROM golang:1.22-alpine AS builder

WORKDIR /app

# Копируем ТОЛЬКО go.mod и go.sum
COPY go.mod go.sum ./

# Качаем зависимости
RUN go mod download

# Копируем код
COPY . .

# Собираем
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

# ---- runtime ----
FROM alpine:latest

WORKDIR /app
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/main .

CMD ["./main"]