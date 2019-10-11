# Build /go/bin/ipmi_exporter
FROM golang:latest AS builder

RUN go get github.com/soundcloud/ipmi_exporter \
 && cd /go/src/github.com/soundcloud/ipmi_exporter \
 && go build


# Container image
FROM ubuntu:18.04

LABEL maintainer="Aggelos Kolaitis <neoaggelos@gmail.com>"

WORKDIR /

RUN apt-get update \
    && apt-get install freeipmi-tools -y --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /go/bin/ipmi_exporter /ipmi_exporter

EXPOSE 9290

CMD ["/ipmi_exporter", "--config.file", "/config.yml"]
