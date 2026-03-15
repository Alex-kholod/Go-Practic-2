# Этап 1: сборка
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /tasks ./services/tasks/cmd/tasks

# Этап 2: образ для запуска
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /tasks .

EXPOSE 8082

CMD ["./tasks"]