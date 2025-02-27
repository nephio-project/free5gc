#  Copyright 2025 The Nephio Authors.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

ARG BUILDER_IMAGE=golang
ARG BASE_IMAGE=alpine

# See: https://github.com/free5gc/free5gc/wiki/Installation

#
# Builder
#

FROM $BUILDER_IMAGE AS builder
ARG VERSION=3.2.1
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

RUN mkdir --parents ./config/ ./config/TLS/ ./public

# Copy executables
COPY --from=builder /go/src/free5gc/bin/* ./
COPY --from=builder /go/src/free5gc/webconsole/bin/webconsole ./

# Copy static files (webconsole frontend)
COPY --from=builder /go/src/free5gc/webconsole/public ./public

# Copy certificates and config files
COPY --from=builder /go/src/free5gc/config/* ./config/
COPY --from=builder /go/src/free5gc/config/TLS/* ./config/TLS/
