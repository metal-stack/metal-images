ARG BASE_OS_NAME
ARG BASE_OS_VERSION

FROM golang:1.20-bullseye as ignition-builder
ARG IGNITION_BRANCH
WORKDIR /work
RUN set -ex \
 && apt-get update \
 && apt-get install -y libblkid-dev \
 && git clone https://github.com/flatcar/ignition.git \
 && cd ignition \
 && git checkout ${IGNITION_BRANCH} \
 && make \
 && strip bin/amd64/ignition

FROM ${BASE_OS_NAME}:${BASE_OS_VERSION}
# Beware, ARGs for ENV variables except FROM line must be below FROM

ARG FRR_VERSION
ARG FRR_VERSION_DETAIL
ARG GOLLDPD_VERSION
ARG DOCKER_APT_OS
ARG DOCKER_APT_CHANNEL
ARG FRR_APT_CHANNEL
ARG CRI_VERSION
ARG SEMVER_MAJOR_MINOR
ARG KERNEL_VERSION
ARG UBUNTU_MAINLINE_KERNEL_VERSION
ARG ICE_VERSION
ARG ICE_PKG_VERSION

ENV DEBCONF_NONINTERACTIVE_SEEN="true" \
    DEBIAN_FRONTEND="noninteractive" \
    DOCKER_URL=https://download.docker.com \
    SEMVER_MAJOR_MINOR=${SEMVER_MAJOR_MINOR} \
    UBUNTU_MAINLINE_KERNEL_VERSION=${UBUNTU_MAINLINE_KERNEL_VERSION}

COPY context/etc/initramfs-tools/conf.d/ /etc/initramfs-tools/conf.d/
COPY --from=ignition-builder /work/ignition/bin/amd64/ignition* /usr/local/bin/
COPY context/kernel-installation.sh /

RUN set -ex \
 && rm -f /.dockerenv \
 && apt-get update \
 && apt-get upgrade --yes \
 && apt-get install --yes --no-install-recommends \
    apparmor-utils \
    apt-transport-https \
    apt-utils \
    ca-certificates \
    cloud-init \
    cpufrequtils \
    curl \
    dbus \
    efibootmgr \
    gettext-base \
    gnupg2 \
    gpg-agent \
    grub-efi-amd64-bin \
    grub2 \
    haveged \
    initramfs-tools \
    ipmitool \
    iptables \
    iputils-ping \
    iproute2 \
    jq \
    lsb-release \
    locales \
    lvm2 \
    mdadm \
    nvme-cli \
    openssh-server \
    procps \
    socat \
    software-properties-common \
    sudo \
    systemd \
    systemd-sysv \
    systemd-timesyncd \
    tcpdump \
    traceroute \
    tzdata \
    watchdog \
    wget \
    vim \
    zstd \
 && curl -fLsS https://github.com/metal-stack/go-lldpd/releases/download/${GOLLDPD_VERSION}/go-lldpd.tgz -o /tmp/go-lldpd.tgz \
 && tar -xf /tmp/go-lldpd.tgz \
 && curl -fLsS ${DOCKER_URL}/linux/${DOCKER_APT_OS}/gpg | apt-key add - \
 && echo "deb [arch=amd64] ${DOCKER_URL}/linux/${DOCKER_APT_OS} ${DOCKER_APT_CHANNEL} stable" > /etc/apt/sources.list.d/docker.list \
 && apt-get update \
 && apt-get install --yes --no-install-recommends docker-ce \
 # Install crictl to be able to manipulate containers managed with containerd instead of dockerd
 && curl -fLsS https://github.com/kubernetes-sigs/cri-tools/releases/download/${CRI_VERSION}/crictl-${CRI_VERSION}-linux-amd64.tar.gz -o /tmp/crictl-${CRI_VERSION}-linux-amd64.tar.gz \
 && tar -xf /tmp/crictl-${CRI_VERSION}-linux-amd64.tar.gz -C /usr/local/bin \
 && echo "LANG=\"en_US.UTF-8\"" > /etc/default/locale \
 && cp /usr/share/zoneinfo/Europe/Berlin /etc/localtime \
 # Switch to use iptables-legacy on worker nodes
 && update-alternatives --set iptables /usr/sbin/iptables-legacy \
 && update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy \
 # install frr from frrouting debian package repo
 && curl -fLsS https://deb.frrouting.org/frr/keys.asc | apt-key add - \
 && echo "deb [arch=amd64] https://deb.frrouting.org/frr ${FRR_APT_CHANNEL} ${FRR_VERSION}" > /etc/apt/sources.list.d/frr.list \
 && apt update \
 && apt install --yes --no-install-recommends frr=${FRR_VERSION_DETAIL} frr-pythontools=${FRR_VERSION_DETAIL} \
 # Install Intel Firmware for e800 based network cards
 && curl -fLsS https://sourceforge.net/projects/e1000/files/ice%20stable/${ICE_VERSION}/ice-${ICE_VERSION}.tar.gz/download -o ice.tar.gz \
 && tar -xf ice.tar.gz ice-${ICE_VERSION}/ddp/ice-${ICE_PKG_VERSION}.pkg \
 && mkdir -p /lib/firmware/intel/ice/ddp/ \
 && mv ice-${ICE_VERSION}/ddp/ice-${ICE_PKG_VERSION}.pkg /lib/firmware/intel/ice/ddp/ice.pkg \
 && rm -rf ice* \
 # install kernel and stuff
 && /kernel-installation.sh \
 # remove dhclient which could lead to dangerous configurations
 && apt-get remove --yes isc-dhcp-client \
 # restore a fake dhclient binary to make networking.service happy
 && ln -sf /bin/true /usr/sbin/dhclient \
 # Remove devel and orphaned packages
 && apt autoremove --yes \
 && rm -rf /tmp/* \
 # create directory to store metal-networker tmp files
 && mkdir -p /etc/metal/networker

RUN echo "---" >> /etc/metal/build-meta.yaml \
 && echo "ignitionVersion: ${IGNITION_BRANCH}" >> /etc/metal/build-meta.yaml \
 && echo "golldpdVersion: ${GOLLDPD_VERSION}" >> /etc/metal/build-meta.yaml \
 && echo "frrVersion: ${FRR_VERSION}" >> /etc/metal/build-meta.yaml

COPY context/ /

# Copy the dependencies of `cloud-init-custom.service` into the image.
COPY /cloud-init/cloud-init-custom.service /etc/systemd/system/
COPY /cloud-init/cloud-init-custom.sh /etc/metal/

# Enable all required systemd services
RUN systemctl set-default multi-user.target \
 && systemctl enable lldpd.service \
                     docker.service \
                     frr.service \
                     systemd-networkd \
                     systemd-timesyncd \
                     watchdog.service \
                     cloud-init-custom.service \
                     serial-getty@ttyS1.service \
 && systemctl disable console-getty.service \
 && systemctl mask console-getty.service \
 && update-ca-certificates \
 # generate locale
 && locale-gen
