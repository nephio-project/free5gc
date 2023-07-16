ARG BUILDER_IMAGE=golang
ARG BASE_IMAGE=alpine

# See: https://github.com/free5gc/free5gc/wiki/Installation

#
# Builder
#

FROM $BUILDER_IMAGE AS builder
ARG VERSION=3.3.0
ENV VERSION=$VERSION

# Note: "golang" builder image is based on Debian
RUN curl --silent --show-error https://dl.yarnpkg.com/debian/pubkey.gpg | gpg --dearmor > /etc/apt/trusted.gpg.d/yarn.gpg
RUN echo "deb https://dl.yarnpkg.com/debian/ stable main" > /etc/apt/sources.list.d/yarn.list
RUN apt-get update
RUN apt-get install --assume-yes nodejs yarn

RUN cd "$GOPATH/src" \
    && git -c advice.detachedHead=false clone --branch "v$VERSION" --recursive --jobs "$(nproc)" https://github.com/free5gc/free5gc.git

RUN cd "$GOPATH/src/free5gc" \
    && make

RUN cd "$GOPATH/src/free5gc" \
    && make webconsole

#
# Image
#

FROM $BASE_IMAGE

WORKDIR /free5gc

RUN mkdir --parents ./cert/ ./config/ ./cert/ ./public

# Copy executables
COPY --from=builder /go/src/free5gc/bin/* ./
COPY --from=builder /go/src/free5gc/webconsole/bin/webconsole ./

# Copy static files (webconsole frontend)
COPY --from=builder /go/src/free5gc/webconsole/public ./public

# Copy certificates and config files
COPY --from=builder /go/src/free5gc/config/* ./config/
COPY --from=builder /go/src/free5gc/cert/* ./cert/
