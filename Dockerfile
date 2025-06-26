FROM golang:bullseye

WORKDIR /opt/falkie

COPY go.mod go.sum ./
RUN go mod download

RUN touch falkie.db

COPY . .
RUN go build -o ./bin/main ./src/main.go

RUN chmod +x ./bin/main

CMD ["./bin/main"]