FROM golang:1.21 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /forwarder main.go

FROM alpine:latest
WORKDIR /
COPY --from=builder /forwarder /forwarder
EXPOSE 8080
ENTRYPOINT ["/forwarder"]
