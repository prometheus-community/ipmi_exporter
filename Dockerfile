ARG ARCH="amd64"
ARG OS="linux"
FROM debian:bullseye-slim
#FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
RUN apt-get update && apt-get install -y freeipmi
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/ipmi_exporter /bin/ipmi_exporter

EXPOSE      9290
USER        nobody
ENTRYPOINT  [ "/bin/ipmi_exporter" ]
