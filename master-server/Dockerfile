# Stage 1: Build
FROM golang:1.23-alpine AS build

WORKDIR /app

# Install git if modules need it
RUN apk add --no-cache git bash

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source
COPY . .

# Build the binary
RUN go build -o server ./cmd

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Copy binary from build stage
COPY --from=build /app/server .

EXPOSE 80
USER appuser

CMD ["./server"]
