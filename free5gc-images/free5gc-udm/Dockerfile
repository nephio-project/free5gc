ARG VERSION=3.2.1
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

WORKDIR /free5gc
RUN mkdir --parents ./log/ ./config/TLS/ ./udm/

# Copy executable and default certs
COPY --from=builder /free5gc/udm ./udm
COPY --from=builder /free5gc/config/TLS/udm.pem ./config/TLS/
COPY --from=builder /free5gc/config/TLS/udm.key ./config/TLS/
COPY --from=builder /free5gc/config/udmcfg.yaml ./config/

VOLUME [ "/free5gc/config" ]
#VOLUME [ "/free5gc/config/TLS" ]

WORKDIR /free5gc/udm
