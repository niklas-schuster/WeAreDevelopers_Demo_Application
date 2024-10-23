# Start from the official Go image
FROM golang:1.23-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o /todo-app

# Start a new stage from scratch
FROM alpine:latest

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /todo-app /todo-app

# Copy static files
COPY --from=builder /app/static /static

# Expose port 9090 to the outside world
EXPOSE 9090

# Command to run the executable
CMD ["/todo-app"]
