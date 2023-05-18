ARG VERSION=3.2.1
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

WORKDIR /free5gc
RUN mkdir --parents ./log/ ./config/ ./TLS/ ./amf/

# Copy executable and default certs/configs
COPY --from=builder /free5gc/amf ./amf
COPY --from=builder /free5gc/config/TLS/amf.pem ./config/TLS/
COPY --from=builder /free5gc/config/TLS/amf.key ./config/TLS/
COPY --from=builder /free5gc/config/amfcfg.yaml ./config/

VOLUME [ "/free5gc/config" ]
#VOLUME [ "/free5gc/config/TLS" ]

WORKDIR /free5gc/amf
