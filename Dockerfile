# Build stage
FROM golang:1.23-alpine AS builder

# Install git for go mod download
WORKDIR /app
RUN apk add --no-cache git

# Download dependencies first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o hclsemver ./cmd/hclsemver

# Final stage - using scratch (empty) image
FROM scratch
WORKDIR /app
COPY --from=builder /app/hclsemver .

# Run the binary
ENTRYPOINT ["/app/hclsemver"]
