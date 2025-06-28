FROM golang:bullseye AS builder

WORKDIR /opt/king

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ./bin/main ./src/main.go

FROM debian:bullseye-slim

WORKDIR /opt/king

# Install dependencies needed for CGO libraries and HTTP requests (ca-certificates)
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \ 
        libc6 \
        libstdc++6 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /opt/king/bin/main ./bin/main

CMD ["./bin/main"]
