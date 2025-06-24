FROM golang:bullseye

RUN go env -w CGO_ENABLED=1

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ./bin src/main.go

CMD ["./bin/main"]