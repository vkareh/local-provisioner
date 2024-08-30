# Define Kubernetes namespace and deployment details
NAMESPACE := local-provisioner

# Define variables
EXTERNAL_IMAGE_REGISTRY := default-route-openshift-image-registry.apps.meer8.vkareh.net
INTERNAL_MAGE_REGISTRY := image-registry.openshift-image-registry.svc:5000
IMAGE_NAME := $(NAMESPACE)/local-provisioner
# IMAGE_TAG := $(shell sha256sum local-provisioner.go | head -c6)
IMAGE_TAG := $(shell date +%s)
DOCKERFILE := Dockerfile

# Define tools
go := go
podman := podman
oc := oc

# .PHONY targets
.PHONY: build test lint clean fmt deps rebuild image push project template deploy all

build:
	$(go) build -o local-provisioner .

test:
	$(go) test ./...

lint:
	golangci-lint run

clean:
	$(go) clean
	rm -f local-provisioner

fmt:
	$(go) fmt ./...

deps:
	$(go) mod tidy

rebuild: clean build

project:
	$(oc) new-project "$(NAMESPACE)" || $(oc) project "$(NAMESPACE)" || true

image: build
	$(podman) build -t "$(EXTERNAL_IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)" -f "$(DOCKERFILE)" .

push: image project
	$(podman) push "$(EXTERNAL_IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)"

template:
	$(oc) process \
		--filename="templates/deployment.yaml" \
		--local="true" \
		--ignore-unknown-parameters="true" \
		--param="NAMESPACE=$(NAMESPACE)" \
		--param="IMAGE_REGISTRY=$(INTERNAL_MAGE_REGISTRY)" \
		--param="IMAGE_NAME=$(IMAGE_NAME)" \
		--param="IMAGE_TAG=$(IMAGE_TAG)" \
	> "templates/deployment.json"

deploy: push project template
	$(oc) apply -f "templates/deployment.json" -n $(NAMESPACE)

all: build
