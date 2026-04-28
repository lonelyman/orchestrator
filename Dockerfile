# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o orchestrator ./cmd/server

# Stage 2: Run
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/orchestrator .

EXPOSE 50000

CMD ["./orchestrator"]