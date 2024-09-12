FROM golang:1.23-alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /shortr ./cmd/main.go

FROM alpine:latest

WORKDIR /

COPY --from=builder /shortr /shortr

EXPOSE 8080

CMD ["/shortr"]