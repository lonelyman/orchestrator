# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o orchestrator ./cmd/server

# Stage 2: Run
FROM alpine:latest

# PDF text extraction and PDF-to-image rendering for Ollama OCR.
RUN apk add --no-cache poppler-utils

WORKDIR /app

COPY --from=builder /app/orchestrator .
COPY prompts ./prompts

EXPOSE 50000

CMD ["./orchestrator"]
