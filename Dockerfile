ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /briefme ./cmd/briefme

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /briefme /usr/local/bin/briefme
COPY config.yaml /etc/briefme/config.yaml
EXPOSE 8080
CMD ["briefme", "serve", "-config", "/etc/briefme/config.yaml"]
