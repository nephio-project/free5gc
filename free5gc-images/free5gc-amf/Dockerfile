ARG VERSION=3.3.0
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

WORKDIR /free5gc
RUN mkdir --parents ./cert/ ./log/ ./config/ ./amf/

# Copy executable and default certs/configs
COPY --from=builder /free5gc/amf ./amf
COPY --from=builder /free5gc/cert/amf.pem ./cert/
COPY --from=builder /free5gc/cert/amf.key ./cert/
COPY --from=builder /free5gc/config/amfcfg.yaml ./config/

VOLUME [ "/free5gc/config" ]


WORKDIR /free5gc/amf
