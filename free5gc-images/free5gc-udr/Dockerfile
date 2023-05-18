ARG VERSION=3.2.1
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

WORKDIR /free5gc
RUN mkdir --parents ./config/ ./log/ ./config/TLS/ ./udr/

# Copy executable and default certs
COPY --from=builder /free5gc/udr ./udr
COPY --from=builder /free5gc/config/TLS/udr.pem ./config/TLS/
COPY --from=builder /free5gc/config/TLS/udr.key ./config/TLS/
COPY --from=builder /free5gc/config/udrcfg.yaml ./config/

VOLUME [ "/free5gc/config" ]
#VOLUME [ "/free5gc/config/TLS" ]

WORKDIR /free5gc/udr
