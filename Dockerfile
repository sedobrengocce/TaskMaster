# Base stage for Go dependencies
FROM golang:1.26-alpine AS base
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache \
    make \
    gcc \
    musl-dev \
    git \
    curl

# Development stage
FROM base AS develop

# Install Air for Go hot reload
RUN go install github.com/air-verse/air@latest

# Copy dependency files first
COPY go.* ./
COPY Makefile ./

# Copy .env if it exists
RUN --mount=type=bind,source=.,target=/context \
    if [ -f /context/.env ]; then cp /context/.env .; fi

# Install dependencies and create directories
RUN make install-tools

# Copy the rest of the application
COPY . .

# Start development servers with file watching
CMD ["make", "dev"]

# Build stage
FROM base AS builder

# Copy dependency files first
COPY go.* ./
COPY Makefile ./

# Copy .env if it exists
RUN --mount=type=bind,source=.,target=/context \
    if [ -f /context/.env ]; then cp /context/.env .; fi

# Install dependencies
RUN make install-tools

# Copy the rest of the application
COPY . .

# Build the application
RUN make build

# Production stage
FROM alpine:3.19 AS production
WORKDIR /app

# Copy built artifacts
COPY --from=builder /app/api .
COPY --from=builder /app/.env* ./

EXPOSE 3000
CMD ["./taskmaster"]
