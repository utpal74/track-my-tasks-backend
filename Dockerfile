
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

# Set environment variable at build stage
# ENV ENV=production
# ENV MONGO_DATABASE=task_tracker
# ENV MONGO_URI=mongodb://admin:password@mongodb:27017/task_tracker?authSource=admin
# ENV REDIS_ADDRESS=redis:6379
# ENV APP_PORT=8082
# ENV ALLOWED_ORIGINS=http://localhost:5173
ENV ENV=production
ENV MONGO_DATABASE=task_tracker
ENV MONGO_URI=mongodb+srv://utpalkumar74:Utpwd4mongo30%4092@cluster0.ufka7.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0&tls=true&tlsInsecure=true
ENV REDIS_ADDRESS=172.31.13.2:6379
ENV APP_PORT=8082
ENV ALLOWED_ORIGINS=*.trackmytasks.net

# Build the Go app
RUN go build -o main .

# Stage 2: Use a base image for runtime (e.g., busybox for debugging or scratch for minimalism)
FROM busybox

# Set the working directory
WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main /main

# Copy the .env file into the container
# COPY --from=builder /app/.env /app/.env


# Set the environment variables again
ENV ENV=production
ENV MONGO_DATABASE=task_tracker
ENV MONGO_URI=mongodb+srv://utpalkumar74:Utpwd4mongo30%4092@cluster0.ufka7.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0&tls=true&tlsInsecure=true
ENV REDIS_ADDRESS=172.31.13.2:6379
ENV APP_PORT=8082
ENV ALLOWED_ORIGINS=*.trackmytasks.net

# Expose port 8082 to the outside world
EXPOSE 8082

# Command to run the executable
ENTRYPOINT ["/main"]
