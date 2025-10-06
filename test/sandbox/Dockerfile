FROM debian:12

ARG CLOUD_HYPERVISOR_VERSION=v48.0
ARG CLOUD_HYPERVISOR_URL=https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/${CLOUD_HYPERVISOR_VERSION}/cloud-hypervisor-static
ADD --chmod=755 "${CLOUD_HYPERVISOR_URL}" /usr/local/bin/cloud-hypervisor

WORKDIR /work