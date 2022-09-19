
FROM golang:1.16-alpine as builder

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates tzdata alpine-sdk && update-ca-certificates

ENV CGO_ENABLED=0
# define RELEASE=1 to hide commit hash
ARG RELEASE=0

WORKDIR /build

# install go tools and cache modules
COPY . .

RUN apk add --no-cache curl bash \
    && make build

FROM docker.io/library/alpine:3.16 as runtime


COPY --from=builder /build/fronius-exporter /usr/bin/
USER 1000:0

ENTRYPOINT ["/usr/local/bin/fronius-exporter"]
