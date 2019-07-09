# Override the default common all.
.PHONY: all
all: precheck style unused build test

include Makefile.common

DOCKER_IMAGE_NAME ?= ipmi-exporter
DOCKER_REPO       ?= soundcloud

