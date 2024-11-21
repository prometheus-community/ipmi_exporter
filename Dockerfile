ARG ARCH="amd64"
ARG OS="linux"
FROM --platform=${OS}/${ARCH} alpine:3
RUN apk --no-cache add freeipmi
LABEL maintainer="Asama.Ai"

ARG ARCH="amd64"
ARG OS="linux"
COPY ./ipmi_exporter /bin/ipmi_exporter

EXPOSE      9290
USER        nobody
ENTRYPOINT  [ "/bin/ipmi_exporter" ]
