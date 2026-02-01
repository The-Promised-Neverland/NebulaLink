# Stage 1:Build
FROM golang:1.21-apline AS build

# Set working directory
WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source
COPY . .

# Build the binary
RUN go build -o server ./cmd

# Stage 2: Runtime
FROM apline:latest

WORKDIR /app

# Copy binary from build stage
COPY --from=build /app/server .

# Expose backend port
EXPOSE 80

# Run as non-root for safety
USER 1000:1000

# Start backend
CMD ["./server"]