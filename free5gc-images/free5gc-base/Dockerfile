ARG BUILDER_IMAGE=golang
ARG BASE_IMAGE=alpine

#
# Builder
#

FROM $BUILDER_IMAGE AS builder
ARG VERSION=3.2.1
ENV VERSION=$VERSION

# Based on Debian Bullseye
# See: https://github.com/free5gc/free5gc/wiki/Installation

RUN curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add -
RUN echo "deb https://dl.yarnpkg.com/debian/ stable main" > /etc/apt/sources.list.d/yarn.list
RUN apt update
RUN apt install --assume-yes nodejs yarn

RUN cd $GOPATH/src \
    && git clone --branch v$VERSION --recursive --jobs $(nproc) https://github.com/free5gc/free5gc.git

RUN cd $GOPATH/src/free5gc \
    && make

RUN cd $GOPATH/src/free5gc \
    && make webconsole

#
# Image
#

FROM $BASE_IMAGE

WORKDIR /free5gc

RUN mkdir --parents ./config/ ./config/TLS/ ./public

# Copy executables
COPY --from=builder /go/src/free5gc/bin/* ./
COPY --from=builder /go/src/free5gc/webconsole/bin/webconsole ./

# Copy static files (webconsole frontend)
COPY --from=builder /go/src/free5gc/webconsole/public ./public

# Copy certificates and config files
COPY --from=builder /go/src/free5gc/config/* ./config/
COPY --from=builder /go/src/free5gc/config/TLS/* ./config/TLS/
