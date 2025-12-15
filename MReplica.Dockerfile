# ---------- Build stage ----------
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod terlebih dahulu (cache dependency)
COPY go.mod go.sum ./
RUN go mod download

# Copy seluruh source
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o burungapp

# ---------- Runtime stage ----------
FROM alpine:3.20

WORKDIR /app

# Copy binary dari builder
COPY --from=builder /app/burungapp .

# Expose port (ubah kalau beda)
EXPOSE 8080

# Run app
CMD ["./burungapp"]
