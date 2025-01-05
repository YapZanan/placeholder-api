# Step 1: Build stage
FROM golang:1.23.4-alpine AS build


WORKDIR /app
COPY go.mod go.sum ./
RUN go mod tidy
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o placeholder-server .

# Step 2: Final minimal image using scratch
FROM scratch

# Copy the pre-built binary, font, and swagger docs from the build stage
COPY --from=build /app/placeholder-server /placeholder-server
COPY --from=build /app/NewAmsterdam-Regular.ttf /NewAmsterdam-Regular.ttf
COPY --from=build /app/docs /docs


EXPOSE 5000
CMD ["/placeholder-server"]
