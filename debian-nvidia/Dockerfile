ARG BASE_OS_NAME
ARG BASE_OS_VERSION

FROM ${BASE_OS_NAME}:${BASE_OS_VERSION}

ENV DEBCONF_NONINTERACTIVE_SEEN="true" \
    DEBIAN_FRONTEND="noninteractive" \
    SEMVER_MAJOR_MINOR=${SEMVER_MAJOR_MINOR}

COPY context/ /

RUN set -ex \
 && rm -f /.dockerenv \
 && apt-get remove --yes linux-image-*-amd64 \
 && apt-get update \
 && apt-get upgrade --yes \
 && apt-get install --yes -t bookworm-backports \
    linux-image-amd64 \
    linux-headers-amd64 \
    nvidia-driver \
    firmware-misc-nonfree \
    dkms

RUN set -ex \
 && curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
 && curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
    sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
    sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list \
 && apt-get update \
 && apt-get install --yes nvidia-container-toolkit

