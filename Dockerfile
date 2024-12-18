FROM golang:1.23.4-alpine AS buildstage

RUN apk update && apk add make gcc git curl

# Enable go modules
ENV GO111MODULE=on

#Build ipmi_exporter
WORKDIR /$GOPATH/src/github.com/platinasystems/ipmi_exporter
COPY . .
RUN make precheck style unused build
RUN mv ipmi_exporter /

#Copy the ipmi_expoter binary
FROM alpine:3
RUN apk --no-cache add freeipmi
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"
WORKDIR /
COPY --from=buildstage /ipmi_exporter /

EXPOSE      9290
USER        nobody
ENTRYPOINT  [ "/ipmi_exporter"]
