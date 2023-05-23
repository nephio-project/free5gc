ARG VERSION=3.2.1
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

WORKDIR /free5gc
RUN mkdir --parents ./log/ ./config/TLS/ ./nssf/

# Copy executable and default certs
COPY --from=builder /free5gc/nssf ./nssf
COPY --from=builder /free5gc/config/TLS/nssf.pem ./config/TLS/
COPY --from=builder /free5gc/config/TLS/nssf.key ./config/TLS/
COPY --from=builder /free5gc/config/nssfcfg.yaml ./config/

VOLUME [ "/free5gc/config" ]
#VOLUME [ "/free5gc/config/TLS" ]

WORKDIR /free5gc/nssf
