# Stage 1: Build
FROM golang:1.21-alpine AS build

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

# Copy binary from build stage
COPY --from=build /app/server .

EXPOSE 80
USER 1000:1000

CMD ["./server"]
