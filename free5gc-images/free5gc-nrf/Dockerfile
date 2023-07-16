ARG VERSION=3.3.0
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

WORKDIR /free5gc
RUN mkdir --parents ./cert/ ./log/ ./cert/ ./nrf/

# Copy executable and default certs
COPY --from=builder /free5gc/nrf ./nrf
COPY --from=builder /free5gc/cert/nrf.pem ./cert/
COPY --from=builder /free5gc/cert/nrf.key ./cert/
COPY --from=builder /free5gc/config/nrfcfg.yaml ./config/

VOLUME [ "/free5gc/config" ]


WORKDIR /free5gc/nrf
