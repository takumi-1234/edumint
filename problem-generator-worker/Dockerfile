# Stage 1: Build the Go application
FROM golang:1.22.2-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy only go.mod first to leverage Docker layer caching.
COPY go.mod ./

# Run 'go mod tidy' to generate go.sum.
RUN go mod tidy

# Download all the dependencies.
RUN go mod download

# Copy the rest of the source code into the container.
COPY . .

# Build the application, creating a static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/problem-generator-worker ./cmd/worker

# Stage 2: Create a minimal final image for production
FROM alpine:latest

# Create a non-root user for better security.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set the working directory for the non-root user
WORKDIR /home/appuser

# Copy the built binary from the 'builder' stage.
COPY --from=builder /app/problem-generator-worker .

# Set the user to the non-root user.
USER appuser

# This worker does not need to expose any ports as it communicates
# via the message queue, but this line is kept for clarity.
# EXPOSE 8080 

# Command to run the executable when the container starts.
CMD ["./problem-generator-worker"]