# These are some common variables for Make

IMG_TAG ?= latest

# Image URL to use all building/pushing image targets
DOCKER_IMG ?= docker.io/ccremer/fronius-exporter:$(IMG_TAG)
QUAY_IMG ?= quay.io/ccremer/fronius-exporter:$(IMG_TAG)
