ARG VERSION=3.3.0
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

WORKDIR /free5gc
RUN mkdir --parents ./cert/ ./log/ ./cert/ ./smf/

# Copy executable and default certs
COPY --from=builder /free5gc/smf ./smf
COPY --from=builder /free5gc/cert/smf.pem ./cert/
COPY --from=builder /free5gc/cert/smf.key ./cert/
COPY --from=builder /free5gc/config/smfcfg.yaml ./config/
COPY --from=builder /free5gc/config/uerouting.yaml ./config/

VOLUME [ "/free5gc/config" ]


WORKDIR /free5gc/smf
