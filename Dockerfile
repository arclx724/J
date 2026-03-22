# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o robokaty .

# ── Run stage ─────────────────────────────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

# Runtime deps
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/robokaty .
COPY --from=builder /app/config.env.sample .

# Create downloads dir for temp files
RUN mkdir -p downloads cache

EXPOSE 8080

CMD ["./robokaty"]
