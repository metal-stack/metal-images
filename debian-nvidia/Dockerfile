ARG BASE_OS_NAME
ARG BASE_OS_VERSION
ARG SEMVER_PATCH

FROM ${BASE_OS_NAME}:${BASE_OS_VERSION}${SEMVER_PATCH}

ENV DEBCONF_NONINTERACTIVE_SEEN="true" \
    DEBIAN_FRONTEND="noninteractive" \
    SEMVER_MAJOR_MINOR=${SEMVER_MAJOR_MINOR}

COPY context/ /

RUN set -ex \
 && rm -f /.dockerenv \
 && apt-get update \
 && apt-get remove --yes linux-image-*-amd64 \
 && apt-get install --yes -t bookworm-backports \
    linux-image-amd64 \
    linux-headers-amd64 \
    dkms \
    software-properties-common \
 && curl -fsSL https://developer.download.nvidia.com/compute/cuda/repos/debian12/x86_64/cuda-keyring_1.1-1_all.deb -o cuda-keyring_1.1-1_all.deb \
 && dpkg -i cuda-keyring_1.1-1_all.deb \
 && add-apt-repository contrib \
 && apt-get update \
 && apt-get install --yes \
    cuda-drivers

# For newer versions of cuda, it might be worth looking at:
# https://developer.nvidia.com/cuda-downloads
# and
# https://docs.nvidia.com/cuda/cuda-installation-guide-linux/#switching-between-driver-module-flavors
RUN set -ex \
 && curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
 && curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
    sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
    sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list \
 && apt-get update \
 && apt-get install --yes nvidia-container-toolkit
