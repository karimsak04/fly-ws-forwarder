FROM golang:1.21 as builder
WORKDIR /app
COPY . .
RUN go build -o forwarder main.go

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/forwarder /app/forwarder
EXPOSE 8080
ENTRYPOINT ["/app/forwarder"]