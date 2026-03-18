FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /parser ./main.go

FROM alpine:3.19

COPY --from=builder /parser /parser

ENTRYPOINT ["/parser"]
