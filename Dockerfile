# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download the dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Stage 2: Create the final minimal image using Distroless
FROM gcr.io/distroless/static-debian11

# Set the working directory in the container
WORKDIR /

# Copy the Go binary from the builder stage
COPY --from=builder /app/main .

# Expose the application port (optional, if your app serves on a specific port)
EXPOSE 3000

# Run the Go binary
ENTRYPOINT ["/main"]
