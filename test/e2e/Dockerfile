FROM golang:1.24-trixie AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=e2e-test
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN mkdir -p build/bin && \
    CGO_ENABLED=0 go build \
    -ldflags="-s -w -X github.com/offline-lab/bootconf/cmd/bootconf/commands.Version=${VERSION} -X github.com/offline-lab/bootconf/cmd/bootconf/commands.Commit=${COMMIT} -X github.com/offline-lab/bootconf/cmd/bootconf/commands.BuildTime=${BUILD_TIME}" \
    -o build/bin/bootconf ./cmd/bootconf

FROM debian:trixie-slim

RUN apt-get update \
 && apt-get install --yes --no-install-recommends \
     sudo \
     systemd \
     dropbear \
     wpasupplicant \
     procps \
 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/build/bin/bootconf /usr/local/bin/bootconf

# Mock systemctl so unitrun tests pass without a running systemd
RUN printf '#!/bin/bash\nexit 0\n' > /usr/local/bin/systemctl && chmod +x /usr/local/bin/systemctl

COPY test/e2e/bootconf.yaml /etc/bootconf/test-config.yaml
COPY test/e2e/source-data/ /boot/firmware/config/
COPY test/e2e/run.sh /test/run.sh

ENTRYPOINT ["/bin/bash", "/test/run.sh"]
