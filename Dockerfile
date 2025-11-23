FROM golang:1.25.4-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/pr-reviewer ./cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/pr-reviewer .

COPY --from=builder /app/migrations ./migrations

ENV MIGRATIONS_PATH=/app/migrations

EXPOSE 8080

CMD ["./pr-reviewer"]
