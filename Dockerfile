FROM ubuntu:16.04 as buildstage

# Install git, curl and build required packages
RUN apt-get update -y && apt-get install -y git curl make gcc tar rsync wget

# Download Go and install it to /usr/local/go
RUN curl -s https://storage.googleapis.com/golang/go1.19.10.linux-amd64.tar.gz | tar -v -C /usr/local -xz
ENV PATH $PATH:/usr/local/go/bin

# Enable go modules
ENV GO111MODULE on
RUN go env -w GOPRIVATE=github.com/platinasystems/*

# Enable access to private github repositories
# Token must be passed with --build-arg GITHUB_TOKEN=<value>
ARG GITHUB_TOKEN
RUN git config --global url."https://$GITHUB_TOKEN:x-oauth-basic@github.com/".insteadOf "https://github.com/"

# Populate the module cache based on the go.{mod,sum} files for ipmi_exporter
COPY go.mod go.mod
RUN go mod download

#Build ipmi_exporter
WORKDIR /$GOPATH/src/github.com/platinasystems/ipmi_exporter
COPY . .
RUN make
RUN mv ipmi_exporter /

#Copy the ipmi_expoter binary
ARG ARCH="amd64"
ARG OS="linux"
FROM --platform=${OS}/${ARCH} alpine:3
RUN apk --no-cache add freeipmi
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"
WORKDIR /
COPY --from=buildstage /ipmi_exporter /

EXPOSE      9290
USER        nobody
ENTRYPOINT  [ "/ipmi_exporter"]
