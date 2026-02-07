FROM docker.io/golang:1.25-alpine as builder

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download -x

# Copy the source from the current directory to the Working Directory inside the container
COPY . ./

# Build
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./oxidized-exporter

FROM docker.io/alpine:3.23

# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/oxidized-exporter /oxidized-exporter

ENTRYPOINT ["/oxidized-exporter"]
