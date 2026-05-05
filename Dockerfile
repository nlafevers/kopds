# Build stage
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application as a static binary
# CGO_ENABLED=0 ensures it's statically linked
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kopds ./cmd/kopds/main.go

# Run stage
FROM alpine:3.19

# Add CA certificates for HTTPS requests if needed
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/kopds .

# Create default directories for data and cache
RUN mkdir -p data cache/images

# Expose the default port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["./kopds"]
