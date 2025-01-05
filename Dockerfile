# Step 1: Build stage
FROM golang:1.23.4-alpine AS build

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules manifests to download dependencies
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod tidy

# Copy the source code into the container
COPY . .

# Build the Go app statically linked
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o placeholder-server .

# Step 2: Final minimal image using scratch
FROM scratch

# Copy the pre-built binary and font from the build stage
COPY --from=build /app/placeholder-server /placeholder-server
COPY --from=build /app/NewAmsterdam-Regular.ttf /NewAmsterdam-Regular.ttf

# Expose port 5000
EXPOSE 5000

# Command to run the application
CMD ["/placeholder-server"]
