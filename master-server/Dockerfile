# Stage 1: Build
FROM golang:1.21-alpine AS build

WORKDIR /app

# Copy go modules first (cache dependencies)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (without binaries)
COPY . .

# Build binary in /app/bin
RUN mkdir -p /app/bin && go build -o /app/bin/server ./cmd

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Copy only the binary
COPY --from=build /app/bin/server .

# Expose backend port
EXPOSE 80

# Run as non-root
USER 1000:1000

# Start backend
CMD ["./server"]
