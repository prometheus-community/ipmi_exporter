ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/ipmi_exporter /bin/ipmi_exporter

EXPOSE      9290
USER        nobody
ENTRYPOINT  [ "/bin/ipmi_exporter" ]
