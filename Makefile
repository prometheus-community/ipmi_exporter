# Override the default common all.
.PHONY: all precheck style unused build test
all: precheck style unused build test

DOCKER_ARCHS      ?= amd64
DOCKER_IMAGE_NAME ?= ipmi-exporter
DOCKER_REPO       ?= soundcloud

include Makefile.common

docker: common-docker
