# Stage 1: Build
FROM node:18-alpine AS build

# Set working directory
WORKDIR /app

# Copy package.json and package-lock.json
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy all frontend source code
COPY . .

# Build arguments for Vite environment variables
ARG VITE_API_BASE_URL=http://localhost:8430
ARG VITE_WS_URL=ws://localhost:8430/ws
ARG VITE_WS_PING_INTERVAL=30000
ARG VITE_WS_PONG_TIMEOUT=60000
ARG VITE_WS_RECONNECT_DELAY=3000
ARG VITE_WS_MAX_RECONNECT_ATTEMPTS=10
ARG VITE_WS_ROLE=frontend

# Set environment variables for build
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_WS_URL=$VITE_WS_URL
ENV VITE_WS_PING_INTERVAL=$VITE_WS_PING_INTERVAL
ENV VITE_WS_PONG_TIMEOUT=$VITE_WS_PONG_TIMEOUT
ENV VITE_WS_RECONNECT_DELAY=$VITE_WS_RECONNECT_DELAY
ENV VITE_WS_MAX_RECONNECT_ATTEMPTS=$VITE_WS_MAX_RECONNECT_ATTEMPTS
ENV VITE_WS_ROLE=$VITE_WS_ROLE

# Build the app
RUN npm run build

# Stage 2: Runtime with Nginx
FROM nginx:alpine

# Copy build output to Nginx's html directory
COPY --from=build /app/dist /usr/share/nginx/html

# Expose port 80
EXPOSE 80

# Start Nginx
CMD ["nginx", "-g", "daemon off;"]