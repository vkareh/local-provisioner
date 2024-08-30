FROM golang:1.22 AS builder

ENV SOURCE_DIR=/local-provisioner
WORKDIR $SOURCE_DIR
COPY . $SOURCE_DIR

ENV GOFLAGS=""
RUN make build

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

RUN microdnf update -y && \
    microdnf install -y util-linux && \
    microdnf clean all

COPY --from=builder local-provisioner/local-provisioner /usr/local/bin/

EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/local-provisioner", "server"]
