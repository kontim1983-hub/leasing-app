FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

FROM alpine:latest
WORKDIR /app

# ВАЖНО: Chromium и зависимости
RUN apk --no-cache add \
    ca-certificates \
    chromium \
    chromium-chromedriver \
    nss \
    freetype \
    freetype-dev \
    harfbuzz \
    ttf-freefont \
    font-noto \
    font-noto-emoji \
    wqy-zenhei \
    && fc-cache -f

ENV CHROME_BIN=/usr/bin/chromium-browser \
    CHROME_PATH=/usr/lib/chromium/

COPY --from=builder /app/main .

RUN mkdir -p /app/screenshots_cache && \
    chmod 777 /app/screenshots_cache

RUN chromium-browser --version

EXPOSE 8080
CMD ["./main"]