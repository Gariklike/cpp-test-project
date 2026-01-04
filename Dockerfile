FROM golang:1.21-alpine AS builder

WORKDIR /app

# Копируем зависимости
COPY authorization-server/go.mod authorization-server/go.sum ./
RUN go mod download

# Копируем исходный код
COPY authorization-server/ .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o authorization-server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник
COPY --from=builder /app/authorization-server .

EXPOSE 8080

CMD ["./authorization-server"]