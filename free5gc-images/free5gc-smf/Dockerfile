ARG VERSION=3.2.1
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

WORKDIR /free5gc
RUN mkdir --parents ./log/ ./config/TLS/ ./smf/

# Copy executable and default certs
COPY --from=builder /free5gc/smf ./smf
COPY --from=builder /free5gc/config/TLS/smf.pem ./config/TLS/
COPY --from=builder /free5gc/config/TLS/smf.key ./config/TLS/
COPY --from=builder /free5gc/config/smfcfg.yaml ./config/
COPY --from=builder /free5gc/config/uerouting.yaml ./config/

VOLUME [ "/free5gc/config" ]
#VOLUME [ "/free5gc/config/TLS" ]

WORKDIR /free5gc/smf
