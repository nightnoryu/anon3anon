FROM golang:1.22.5

WORKDIR /app

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/anon3anon ./cmd/anon3anon

CMD ["/app/bin/anon3anon"]
