# Stage 1: Build the Go app
# Use a robust Go image for the build environment.
FROM golang:1.24-alpine AS builder

# Set the Current Working Directory inside the container.
WORKDIR /app

# Copy the Go Modules manifests (go.mod and go.sum) to leverage Docker's layer caching.
COPY go.mod go.sum ./

# Download dependencies. This will only run if go.mod or go.sum change.
RUN go mod download

# Copy the source code, including any configuration files like .env.
COPY . .

# Build the Go application into a single executable binary.
# The "-o main" flag specifies the output file name as 'main'.
RUN go build -o main .

# Stage 2: Create a minimal, production-ready image.
# Use a minimal Alpine Linux image. Busybox is very stripped down and can sometimes
# be missing necessary libraries that even Go binaries might link against. Alpine is a safer choice.
FROM alpine:3.19

# Set the working directory for the application.
WORKDIR /app

# Copy the pre-built binary from the 'builder' stage into the final image.
# We are only copying the single executable, which keeps the final image size small.
COPY --from=builder /app/main .

# Expose port 8080. This is just documentation; the port must also be mapped in docker-compose.yml.
EXPOSE 8080

# The command to run the executable when the container starts.
ENTRYPOINT ["./main"]

# Add a healthcheck to your Dockerfile for production.
# This will allow Docker to monitor if your application is actually up and running.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget -q http://localhost:8080/ || exit 1
