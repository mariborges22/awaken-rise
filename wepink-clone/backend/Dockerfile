# Stage 1: Build
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main cmd/api/main.go

# Stage 2: Security Scan
FROM aquasec/trivy:latest AS scanner
COPY --from=builder /app/main /main
RUN trivy rootfs --exit-code 0 --severity HIGH,CRITICAL /main

# Stage 3: Final Production Image
FROM alpine:3.19
RUN apk add --no-cache ca-certificates

# Create a non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /home/appuser/

# Copy binary from builder
COPY --from=builder /app/main .

# Change ownership to non-root user
RUN chown appuser:appgroup main && chmod +x main

# Switch to non-root user
USER appuser

EXPOSE 8080
CMD ["./main"]
