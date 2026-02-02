# Stage 1: Build
FROM golang:1.21-alpine AS build

WORKDIR /src

# Only copy Go module files first
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary into a separate folder
RUN mkdir -p /out && go build -o /out/server ./cmd

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Copy only the binary
COPY --from=build /out/server .

EXPOSE 80
USER 1000:1000
CMD ["./server"]
