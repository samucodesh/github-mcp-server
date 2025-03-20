FROM golang:1.23.7 AS build
# Set the working directory
WORKDIR /build
# Copy the current directory contents into the working directory
COPY . .
# Install dependencies
RUN go mod download
# Build the server
RUN CGO_ENABLED=0 go build -o github-mcp-server cmd/github-mcp-server/main.go
# Make a stage to run the app
FROM gcr.io/distroless/base-debian12
# Set the working directory
WORKDIR /server
# Copy the binary from the build stage
COPY --from=build /build/github-mcp-server .
# Command to run the server
CMD ["./github-mcp-server", "stdio"]
