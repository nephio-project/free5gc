ARG VERSION=3.3.0
ARG REGISTRY=docker.io/nephio
ARG BASE_IMAGE=alpine

FROM $REGISTRY/free5gc-base:$VERSION AS builder
FROM $BASE_IMAGE

ENV GIN_MODE="release"

RUN apk --no-cache add libc6-compat

WORKDIR /free5gc
RUN mkdir --parents ./cert/ ./config/ ./webconsole/public

# Copy executable and default configs/statics
COPY --from=builder /free5gc/webconsole ./webconsole/
COPY --from=builder /free5gc/config/webuicfg.yaml ./config/
COPY --from=builder /free5gc/public ./webconsole/public

VOLUME [ "/free5gc/config" ]
#VOLUME [ "/free5gc/support/TLS" ]

WORKDIR /free5gc/webconsole
