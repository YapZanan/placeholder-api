# Step 1: Build stage
FROM golang:1.23.4-alpine AS build


WORKDIR /app

# Install necessary tools
# RUN apk add --no-cache upx curl git
RUN apk add --no-cache curl git


# Copy and install Go dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the application code
COPY . .

# Generate Swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN swag init

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags="-s -w" -o placeholder-server


# Compress the binary using UPX extreme*
# RUN upx --best --lzma placeholder-server

# Step 2: Final minimal image using scratch
FROM scratch

# Copy the pre-built binary, font, and Swagger docs from the build stage
COPY --from=build /app/placeholder-server /placeholder-server
COPY --from=build /app/NewAmsterdam-Regular.ttf /NewAmsterdam-Regular.ttf
COPY --from=build /app/docs /docs


EXPOSE 5000
CMD ["/placeholder-server"]
