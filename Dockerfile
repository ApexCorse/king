FROM golang:bullseye

WORKDIR /opt/king

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ./bin/main ./src/main.go

RUN chmod +x ./bin/main

CMD ["./bin/main"]