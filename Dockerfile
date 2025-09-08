FROM golang:1.25.1-alpine AS builder
WORKDIR /
COPY . .
RUN go build -o start_node ./cmd/main.go


FROM alpine:latest
WORKDIR /

COPY --from=builder /start_node .
EXPOSE 8000/udp
ENTRYPOINT ["./start_node"]

