FROM golang:1.18 AS builder

WORKDIR /go/src/app


COPY . .

RUN go mod tidy
RUN go mod download
RUN go build -o main main.go
FROM golang:1.18 AS runner


COPY --from=builder /go/src/app/main .

CMD ["./main"]

EXPOSE 8080