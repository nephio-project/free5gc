ARG VERSION=3.3.0
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

WORKDIR /free5gc
RUN mkdir --parents ./cert/ ./config/ ./log/ ./cert/ ./pcf/

# Copy executable and default certs
COPY --from=builder /free5gc/pcf ./pcf
COPY --from=builder /free5gc/cert/pcf.pem ./cert/
COPY --from=builder /free5gc/cert/pcf.key ./cert/
COPY --from=builder /free5gc/config/pcfcfg.yaml ./config/

VOLUME [ "/free5gc/config" ]


WORKDIR /free5gc/pcf
