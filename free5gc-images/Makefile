FREE5GC_VERSION ?= 3.2.1
IMG_REGISTRY ?= docker.io/nephio
BASE_IMAGE ?= alpine:3.18.0
BUILDER_IMAGE ?= golang:1.20.5

ifeq ($(NO_CACHE),true)
	BUILD_ARGS=--no-cache
endif

define build
	docker build \
		--tag "$(IMG_REGISTRY)/$1:$(FREE5GC_VERSION)" \
		--tag "$(IMG_REGISTRY)/$1:latest" \
		--build-arg "VERSION=$(FREE5GC_VERSION)" \
		--build-arg "REGISTRY=$(IMG_REGISTRY)" \
		--build-arg "BASE_IMAGE=$(BASE_IMAGE)" \
		--build-arg "BUILDER_IMAGE=$(BUILDER_IMAGE)" \
		--file "./$1/Dockerfile" . \
		$(BUILD_ARGS)
endef

define push
	docker push "$(IMG_REGISTRY)/$1:$(FREE5GC_VERSION)"
	docker push "$(IMG_REGISTRY)/$1:latest"
endef

.PHONY: all
all: build

.PHONY: build
build:
	$(call build,free5gc-base)
	$(call build,free5gc-amf)
	$(call build,free5gc-ausf)
	$(call build,free5gc-n3iwf)
	$(call build,free5gc-nrf)
	$(call build,free5gc-nssf)
	$(call build,free5gc-pcf)
	$(call build,free5gc-smf)
	$(call build,free5gc-udm)
	$(call build,free5gc-udr)
	$(call build,free5gc-upf)
	$(call build,free5gc-webconsole)

.PHONY: push
push:
	$(call push,free5gc-amf)
	$(call push,free5gc-ausf)
	$(call push,free5gc-n3iwf)
	$(call push,free5gc-nrf)
	$(call push,free5gc-nssf)
	$(call push,free5gc-pcf)
	$(call push,free5gc-smf)
	$(call push,free5gc-udm)
	$(call push,free5gc-udr)
	$(call push,free5gc-upf)
	$(call push,free5gc-webconsole)
