ARG BASE_OS_NAME
ARG BASE_OS_VERSION
ARG FRR_VERSION
ARG METAL_NETWORKER_VERSION

FROM ghcr.io/metal-stack/frr:${FRR_VERSION} AS frr-artifacts
FROM ghcr.io/metal-stack/metal-networker:${METAL_NETWORKER_VERSION} as metal-networker

FROM golang:1.15-buster as ignition-builder
ARG IGNITION_BRANCH
WORKDIR /work
RUN set -ex \
 && apt-get update \
 && apt-get install -y libblkid-dev \
 && git clone https://github.com/coreos/ignition.git \
 && cd ignition \
 && git checkout ${IGNITION_BRANCH} \
 && make

FROM ${BASE_OS_NAME}:${BASE_OS_VERSION}
# Beware, ARGs for ENV variables except FROM line must be below FROM

ARG YQ_VERSION
ARG GOLLDPD_VERSION
ARG METAL_NETWORKER_VERSION
ARG DOCKER_APT_OS
ARG DOCKER_APT_CHANNEL
ARG CRI_VERSION
ARG SEMVER_MAJOR_MINOR
ARG KERNEL_VERSION

ENV DEBCONF_NONINTERACTIVE_SEEN="true" \
    DEBIAN_FRONTEND="noninteractive" \
    YQ_DOWNLOAD=https://github.com/mikefarah/yq/releases/download \
    YQ=/usr/local/bin/yq \
    DOCKER_URL=https://download.docker.com \
    SEMVER_MAJOR_MINOR=${SEMVER_MAJOR_MINOR}

COPY context/etc/initramfs-tools/conf.d/ /etc/initramfs-tools/conf.d/
COPY --from=frr-artifacts /artifacts/*.deb /tmp/
COPY --from=ignition-builder /work/ignition/bin/amd64/ignition* /usr/local/bin/
COPY context/kernel-and-timesyncd-installation.sh /

RUN set -ex \
 && sed 's@archive.ubuntu.com@de.archive.ubuntu.com@' -i /etc/apt/sources.list \
 && sed 's@security.ubuntu.com@de.archive.ubuntu.com@' -i /etc/apt/sources.list \
 && apt-get update \
 && apt-get upgrade --yes \
 && apt-get install --yes --no-install-recommends \
    apparmor-utils \
    apt-transport-https \
    apt-utils \
    ca-certificates \
    cloud-init \
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
    isc-dhcp-client \
    jq \
    lsb-release \
    lvm2 \
    mdadm \
    procps \
    socat \
    software-properties-common \
    sudo \
    systemd \
    systemd-sysv \
    tzdata \
    watchdog \
    wget \
    /tmp/frr_*.deb \
    /tmp/frr-pythontools_*.deb \
    /tmp/libyang*.deb \
    vim \
 && curl -fLsS ${YQ_DOWNLOAD}/${YQ_VERSION}/yq_linux_amd64 -o ${YQ} \
 && chmod +x ${YQ} \
 && curl -fLsS https://github.com/metal-stack/go-lldpd/releases/download/${GOLLDPD_VERSION}/go-lldpd.tgz -o /tmp/go-lldpd.tgz \
 && tar -xf /tmp/go-lldpd.tgz \
 && curl -fLsS ${DOCKER_URL}/linux/${DOCKER_APT_OS}/gpg > docker.key \
 && apt-key add docker.key \
 && rm -f docker.key \
 && add-apt-repository "deb [arch=amd64] ${DOCKER_URL}/linux/${DOCKER_APT_OS} ${DOCKER_APT_CHANNEL} stable" \
 && apt-get update \
 && apt-get install --yes --no-install-recommends docker-ce \
 # docker is always installed in /usr/bin/docker, on ubuntu /bin is a link to /usr/bin
 # debian does not have this link, therefore /bin/docker does not exist.
 # gardener requires /bin/docker in their kubelet.service.
 && ln -s /usr/bin/docker /bin/docker || true \
 # Install crictl to be able to manipulate containers managed with containerd instead of dockerd
 && curl -fLsS https://github.com/kubernetes-sigs/cri-tools/releases/download/${CRI_VERSION}/crictl-${CRI_VERSION}-linux-amd64.tar.gz -o /tmp/crictl-${CRI_VERSION}-linux-amd64.tar.gz \
 && tar -xf /tmp/crictl-${CRI_VERSION}-linux-amd64.tar.gz -C /usr/local/bin \
 && echo "LANG=\"en_US.UTF-8\"" > /etc/default/locale \
 && cp /usr/share/zoneinfo/Europe/Berlin /etc/localtime \
 # Switch to use iptables-legacy on worker nodes
 && update-alternatives --set iptables /usr/sbin/iptables-legacy \
 && update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy \
 && /kernel-and-timesyncd-installation.sh \
 # Remove devel and orphaned packages
 && apt remove --yes \
    libc-dev-bin \
    libc6-dev \
    libpcre3-dev \
    libyang-dev \
    linux-libc-dev \
 && apt autoremove --yes \
 && rm -rf /tmp/*

COPY context/ /

# Copy the dependencies of `cloud-init-custom.service` into the image.
COPY /cloud-init/cloud-init-custom.service /etc/systemd/system/
COPY /cloud-init/cloud-init-custom.sh /etc/metal/

# Set default target to multi-user.target - otherwise it will be graphical.target
# Fix WantedBy= definition of frr.service - network-online
# is a target that is not pulled in per default during startup
RUN systemctl set-default multi-user.target \
 && sed -i "s/^WantedBy=network-online\.target/WantedBy=network\.target/" \
           /lib/systemd/system/frr.service \
 && systemctl enable lldpd.service \
                     docker.service \
                     frr.service \
                     systemd-networkd \
                     systemd-resolved \
                     systemd-timesyncd \
                     watchdog.service \
                     cloud-init-custom.service \
 && update-ca-certificates

# Provide metal-networker that is called during the install routine to setup networking.
COPY --from=metal-networker /metal-networker /etc/metal/networker/
