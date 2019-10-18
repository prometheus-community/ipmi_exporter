# Build /go/bin/ipmi_exporter
FROM quay.io/prometheus/golang-builder:1.13-base AS builder

RUN go get -d github.com/soundcloud/ipmi_exporter \
 && cd /go/src/github.com/soundcloud/ipmi_exporter \
 && make


# Container image
FROM ubuntu:18.04

LABEL maintainer="Aggelos Kolaitis <neoaggelos@gmail.com>"

WORKDIR /

RUN apt-get update \
    && apt-get install freeipmi-tools -y --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /go/src/github.com/soundcloud/ipmi_exporter /bin/ipmi_exporter

EXPOSE 9290

CMD ["/bin/ipmi_exporter", "--config.file", "/config.yml"]
