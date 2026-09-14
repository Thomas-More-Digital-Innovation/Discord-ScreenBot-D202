# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build tools and certificates
RUN apk add --no-cache ca-certificates git

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/screenbot .

# Runtime stage
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/screenbot /usr/local/bin/screenbot

# Entrypoint
ENTRYPOINT ["screenbot"]
CMD ["run", "--config", "/etc/screenbot/config.yaml"]
