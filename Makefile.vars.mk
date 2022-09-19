# This file is managed by greposync.
# Do not modify manually.
# Adjust variables in `.sync.yml`.

# These are some common variables for Make

BIN_FILENAME ?= fronius-exporter

# Image URL to use all building/pushing image targets
IMG_TAG ?= latest
LOCAL_IMG ?= local.dev/ccremer/fronius-exporter:$(IMG_TAG)

PLATFORM ?= linux/amd64
