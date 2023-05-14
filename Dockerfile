# Builder Container
FROM golang:1.20 as builder
RUN mkdir /app
RUN mkdir /out
WORKDIR /app

## Dependencies
COPY go.sum .
COPY go.mod .
RUN go mod download -x

## Compile
COPY . .
RUN CGO_ENABLED=0 go build -v -o /out/inet256 ./cmd/inet256

# Final Container
FROM alpine:3.18
COPY --from=builder /out/inet256 /usr/bin/inet256
# API
EXPOSE 2560:2560
# TRANSPORT
EXPOSE 25632:25632/udp

ENTRYPOINT inet256 daemon --config=/config/config.yml
