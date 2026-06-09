FROM golang:1.24-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/offline-lab/bootconf/internal/version.Version=e2e-test" \
    -o /bootconf cmd/bootconf/main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    sudo \
    jq \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /bootconf /usr/local/bin/bootconf
COPY test/e2e/bootconf.yaml /etc/bootconf/test-config.yaml
COPY test/e2e/source-data/ /boot/firmware/config/
COPY test/e2e/run.sh /run.sh

RUN chmod +x /run.sh
ENTRYPOINT ["/run.sh"]
