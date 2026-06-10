FROM golang:1.24-trixie AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
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
 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/build/bin/bootconf /usr/local/bin/bootconf
COPY bootconf.yaml /etc/bootconf/bootconf.yaml.example

ENTRYPOINT ["bootconf"]
