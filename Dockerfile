FROM docker.io/library/alpine:3.21 as runtime

ENTRYPOINT ["fronius-exporter"]

RUN \
    apk add --no-cache curl bash

COPY fronius-exporter /usr/bin/
USER 1000:0
