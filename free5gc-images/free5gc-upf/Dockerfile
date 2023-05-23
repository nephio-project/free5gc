ARG VERSION=3.2.1
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

# Install UPF dependencies
RUN apk --no-cache add iproute2 iptables

WORKDIR /free5gc
RUN mkdir --parents ./config/ ./upf/

# Copy executable and default configs/certs
COPY --from=builder /free5gc/upf ./upf/
COPY --from=builder /free5gc/config/upfcfg.yaml ./config/

VOLUME [ "/free5gc/config" ]

WORKDIR /free5gc/upf
