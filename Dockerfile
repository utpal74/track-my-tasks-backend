
# Stage 1: Build the Go app
FROM golang:1.22.3 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code including .env file into the container
COPY . .

# Build the Go app
RUN go build -o main .

# Stage 2: Use a base image for runtime (e.g., busybox for debugging or scratch for minimalism)
FROM busybox

# Set the working directory
WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main /main

# Copy the .env file into the container
COPY --from=builder /app/.env /app/.env

# Expose port 8082 to the outside world
EXPOSE 8082

# Command to run the executable
ENTRYPOINT ["/main"]
