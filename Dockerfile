FROM golang:1.22

WORKDIR /app

COPY * ./

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build cmd/anon3anon/main.go -o /anon3anon

CMD ["/anon3anon"]
