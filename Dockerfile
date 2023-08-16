# Use the official Go image to create a build environment
FROM golang:1.19 as builder

# Copy the local package files to the container's workspace
COPY . .

# Set the working directory
WORKDIR /bin/app

# Build the application inside the container
RUN CGO_ENABLED=0 GOOS=linux go build -v -o app

# Use a small image for the final production image
FROM alpine:3.14

# Copy the binary from the builder stage
COPY --from=builder /src/app /app

# Run the application
ENTRYPOINT ["/app"]
