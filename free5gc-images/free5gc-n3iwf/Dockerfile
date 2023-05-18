ARG VERSION=3.2.1
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

# Install N3IWF dependencies
RUN apk add -U iproute2

WORKDIR /free5gc
RUN mkdir --parents ./log/ ./config/TLS/ ./n3iwf/

# Copy executable and default certs/configs
COPY --from=builder /free5gc/n3iwf ./n3iwf
COPY --from=builder /free5gc/config/TLS/n3iwf.pem ./config/TLS/
COPY --from=builder /free5gc/config/TLS/n3iwf.key ./config/TLS/
COPY --from=builder /free5gc/config/n3iwfcfg.yaml ./config/

VOLUME [ "/free5gc/config" ]
#VOLUME [ "/free5gc/config/TLS" ]

WORKDIR /free5gc/n3iwf
