ARG BASE_OS_NAME
ARG BASE_OS_VERSION
ARG FRR_VERSION

FROM metalstack/frr:${FRR_VERSION} AS frr-artifacts

FROM golang:1.13-buster as ignition-builder
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
ARG LINUX_IMAGE
ARG DOCKER_APT_OS
ARG DOCKER_APT_CHANNEL

ENV DEBCONF_NONINTERACTIVE_SEEN="true" \
    DEBIAN_FRONTEND="noninteractive" \
    YQ_DOWNLOAD=https://github.com/mikefarah/yq/releases/download \
    YQ=/usr/local/bin/yq \
    DOCKER_URL=https://download.docker.com

COPY context/etc/initramfs-tools/conf.d/ /etc/initramfs-tools/conf.d/
COPY --from=frr-artifacts /artifacts/*.deb /tmp/
COPY --from=ignition-builder /work/ignition/bin/amd64/ignition* /usr/local/bin/
COPY context/kernel-installation.sh /

RUN set -ex \
 && sed 's@archive.ubuntu.com@de.archive.ubuntu.com@' -i /etc/apt/sources.list \
 && sed 's@security.ubuntu.com@de.archive.ubuntu.com@' -i /etc/apt/sources.list \
 && apt-get update \
 && apt-get install --yes --no-install-recommends \
    apt-transport-https \
    apt-utils \
    ca-certificates \
    curl \
    dbus \
    efibootmgr \
    gettext-base \
    gnupg2 \
    gpg-agent \
    grub-efi-amd64-bin \
    grub2 \
    # Only ifupdown2 support association of more than one IP address to lo LOOPBACK interface.
    ifupdown2 \
    initramfs-tools \
    ipmitool \
    iputils-ping \
    iproute2 \
    isc-dhcp-client \
    jq \
    lsb-release \
    ntp \
    openssh-server \
    procps \
    socat \
    software-properties-common \
    sudo \
    systemd \
    systemd-sysv \
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
 # Ensure that there exists a link from /usr/bin/python to python2
 # This is needed for ifupdown2 since they have /usr/bin/python as interpreter: https://github.com/CumulusNetworks/ifupdown2/blob/master/ifupdown2/__main__.py#L1
 # - with Ubuntu >= 20.04 this link does not exist anymore
 # - Debian Testing removed ifupdown2 from the repo: https://tracker.debian.org/news/1064997/ifupdown2-removed-from-testing/
 # - but a python3 based ifupdown2 version is on the way: https://github.com/CumulusNetworks/ifupdown2/issues/126
 && ln -sf /usr/bin/python2 /usr/bin/python \
 && /kernel-installation.sh \
 && rm -rf /tmp/*

COPY context/ /
RUN /install-fits-ca.sh

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
                     watchdog.service \
 && update-ca-certificates

# Provide metal-networker that is called during the install routine to setup networking.
WORKDIR /etc/metal/networker
RUN curl -fLOsS https://github.com/metal-stack/metal-networker/releases/download/${METAL_NETWORKER_VERSION}/metal-networker.tgz \
 && tar xzf metal-networker.tgz \
 && rm metal-networker.tgz
