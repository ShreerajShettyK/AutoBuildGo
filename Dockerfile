# Use the official Golang image as the base image for the build
FROM golang:1.18-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .
COPY .env /app/.env

# Install git
RUN apk add --no-cache git

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./main.go

# Use a minimal image as the base image for the final container
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /root/

# Copy the pre-built binary file from the builder stage
COPY --from=builder /app/main .

# Copy git and other necessary tools into the final image
RUN apk add --no-cache git curl

# Copy the entrypoint script into the container
COPY entrypoint.sh /entrypoint.sh

# Make the entrypoint script executable
RUN chmod +x /entrypoint.sh

# Expose port 8082 to the outside world
EXPOSE 8082

# Set the entrypoint to the script
ENTRYPOINT ["/entrypoint.sh"]
