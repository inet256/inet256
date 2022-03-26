FROM golang:1.18 as builder
RUN mkdir /app
RUN mkdir /out
WORKDIR /app

# Dependencies
COPY go.sum .
COPY go.mod .
RUN go mod download -x

# Compile
COPY . .
RUN CGO_ENABLED=0 go build -v -o /out/inet256 ./cmd/inet256

FROM alpine:3.14
COPY --from=builder /out/inet256 /usr/bin/inet256

