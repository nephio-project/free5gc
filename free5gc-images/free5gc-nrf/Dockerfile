ARG VERSION=3.2.1
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

WORKDIR /free5gc
RUN mkdir --parents ./log/ ./config/TLS/ ./nrf/

# Copy executable and default certs
COPY --from=builder /free5gc/nrf ./nrf
COPY --from=builder /free5gc/config/TLS/nrf.pem ./config/TLS/
COPY --from=builder /free5gc/config/TLS/nrf.key ./config/TLS/
COPY --from=builder /free5gc/config/nrfcfg.yaml ./config/

VOLUME [ "/free5gc/config" ]
#VOLUME [ "/free5gc/config/TLS" ]

WORKDIR /free5gc/nrf
