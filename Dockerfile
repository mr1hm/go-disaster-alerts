# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /disaster-alert ./cmd/disaster-alert

# Runtime stage
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache tzdata && mkdir -p /app/data

COPY --from=builder /disaster-alert .

EXPOSE 8080 50051

CMD ["./disaster-alert"]
