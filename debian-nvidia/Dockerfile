FROM baseapp

ENV DEBCONF_NONINTERACTIVE_SEEN="true" \
    DEBIAN_FRONTEND="noninteractive"

COPY --from=ctx . /

RUN set -ex \
 && rm -f /.dockerenv \
 && apt-get update \
 && apt-get install --yes \
    dkms \
 && curl -fsSL https://developer.download.nvidia.com/compute/cuda/repos/debian12/x86_64/cuda-keyring_1.1-1_all.deb -o cuda-keyring_1.1-1_all.deb \
 && dpkg -i cuda-keyring_1.1-1_all.deb \
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
