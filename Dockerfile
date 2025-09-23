FROM golang:1.25.1-alpine AS builder
WORKDIR /
COPY . .
RUN go build -o node ./cmd/main.go


FROM alpine:latest
WORKDIR /

COPY --from=builder /node .
ENTRYPOINT ["./node"]

