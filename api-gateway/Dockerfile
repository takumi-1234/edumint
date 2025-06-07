# Stage 1: Build the Go application
FROM golang:1.22.2-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy only go.mod first to leverage Docker layer caching.
# This step is only re-run if go.mod changes.
COPY go.mod ./

# Run 'go mod tidy' to generate go.sum and remove unused dependencies.
# This ensures that a go.sum file is not required in the local source directory.
RUN go mod tidy

# Download all the dependencies from the internet and cache them.
RUN go mod download

# Copy the rest of the source code into the container.
COPY . .

# Build the application, creating a static binary.
# CGO_ENABLED=0 is important for creating a static binary without C dependencies.
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/api-gateway ./cmd/server

# Stage 2: Create a minimal final image for production
FROM alpine:latest

# It's a good practice to run containers as a non-root user.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set the working directory for the non-root user
WORKDIR /home/appuser

# Copy the built binary from the 'builder' stage.
COPY --from=builder /app/api-gateway .

# Set the user to the non-root user created above.
USER appuser

# Expose port 8080 to the outside world.
EXPOSE 8080

# Command to run the executable when the container starts.
CMD ["./api-gateway"]