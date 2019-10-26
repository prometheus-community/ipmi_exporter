# Build /go/bin/ipmi_exporter
FROM quay.io/prometheus/golang-builder:1.13-base AS builder
ADD . /go/src/github.com/soundcloud/ipmi_exporter/
RUN cd /go/src/github.com/soundcloud/ipmi_exporter && make

# Container image
FROM ubuntu:18.04
WORKDIR /
RUN apt-get update \
    && apt-get install freeipmi-tools -y --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /go/src/github.com/soundcloud/ipmi_exporter/ipmi_exporter /bin/ipmi_exporter

EXPOSE 9290
ENTRYPOINT ["/bin/ipmi_exporter"]
CMD ["--config.file", "/config.yml"]
