ARG BASE_OS_VERSION

FROM golang:1.18-buster as ignition-builder
ARG IGNITION_BRANCH
WORKDIR /work
RUN set -ex \
 && apt-get update \
 && apt-get install -y libblkid-dev \
 && git clone https://github.com/coreos/ignition.git \
 && cd ignition \
 && git checkout ${IGNITION_BRANCH} \
 && GO111MODULE=off make \
 && strip bin/amd64/ignition

FROM centos:${BASE_OS_VERSION}
# Beware, ARGs for ENV variables except FROM line must be below FROM

ARG YQ_VERSION
ARG GOLLDPD_VERSION
ARG METAL_NETWORKER_VERSION
ARG FRR_VERSION

ENV YQ_DOWNLOAD=https://github.com/mikefarah/yq/releases/download \
    YQ=/usr/local/bin/yq

COPY context/etc/initramfs-tools/conf.d/ /etc/initramfs-tools/conf.d/
COPY --from=ignition-builder /work/ignition/bin/amd64/ignition* /usr/local/bin/

RUN set -ex \
 && yum install -y \
    ca-certificates \
    chrony \
    curl \
    dbus \
    efibootmgr \
    gettext \
    gnupg2 \
    grub2 \
    grub2-efi \
    grub2-efi-modules \
    # Only ifupdown2 support association of more than one IP address to lo LOOPBACK interface.
    # ifupdown2 \
    ipmitool \
    jq \
    kernel \
    mdadm \
    openssl \
    openssh-server \
    passwd \
    procps \
    shim \
    socat \
    sudo \
    systemd \
    systemd-networkd \
    systemd-sysv \
    watchdog \
    wget \
    vim \
    yum-utils
RUN curl -fLsS ${YQ_DOWNLOAD}/${YQ_VERSION}/yq_linux_amd64 -o ${YQ} \
 && chmod +x ${YQ} \
 && curl -fLsS https://github.com/metal-stack/go-lldpd/releases/download/${GOLLDPD_VERSION}/go-lldpd.tgz -o /tmp/go-lldpd.tgz \
 && tar -xf /tmp/go-lldpd.tgz \
 && yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo \
 && yum install -y docker-ce docker-ce-cli containerd.io \
 && echo "LANG=\"en_US.UTF-8\"" > /etc/default/locale \
 && cp /usr/share/zoneinfo/Europe/Berlin /etc/localtime \
 && rm -rf /tmp/*

# Install FRR see https://rpm.frrouting.org/ for available channels
RUN set -ex \
 && curl -fLsS https://rpm.frrouting.org/repo/${FRR_VERSION}-repo-1-0.el7.noarch.rpm -o /tmp/frr-repo.rpm \
 && yum install -y /tmp/frr-repo.rpm \
 && yum install -y frr frr-pythontools

# Install jq because centos:7 does not have a package
RUN set -ex \
 && curl -fLsS https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64 -o /usr/bin/jq \
 && chmod +x /usr/bin/jq

RUN echo "---" >> /etc/metal/build-meta.yaml \
 && echo "ignitionVersion: ${IGNITION_BRANCH}" >> /etc/metal/build-meta.yaml \
 && echo "golldpdVersion: ${GOLLDPD_VERSION}" >> /etc/metal/build-meta.yaml \
 && echo "frrVersion: ${FRR_VERSION}" >> /etc/metal/build-meta.yaml \
 && echo "osName: ${SEMVER}" >> /etc/metal/build-meta.yaml \
 && echo "osSemver: ${BASE_OS_VERSION}" >> /etc/metal/build-meta.yaml \
 && echo "baseOSVersion: ${BASE_OS_VERSION}" >> /etc/metal/build-meta.yaml

COPY context/ /
# Set default target to multi-user.target - otherwise it will be graphical.target
# Fix WantedBy= definition of frr.service - network-online
# is a target that is not pulled in per default during startup
# FIXME systemctl enable can be remove if the preset mechanism works
# in /usr/lib/systemd/system-preset/metal.preset
RUN systemctl set-default multi-user.target \
 && systemctl enable lldpd.service \
                     docker.service \
                     frr.service \
                     systemd-networkd \
                     watchdog.service \
 && update-ca-trust

# Provide metal-networker that is called during the install routine to setup networking.
WORKDIR /etc/metal/networker
RUN curl -fLOsS https://github.com/metal-stack/metal-networker/releases/download/${METAL_NETWORKER_VERSION}/metal-networker.tgz \
 && tar xzf metal-networker.tgz \
 && rm metal-networker.tgz /etc/sysconfig/network-scripts/ifcfg-lo

# execute dracut to regenerate initramfs with all drivers because otherwise only
# a few drivers are included and the kexec into centos will fail because no disks where found
WORKDIR /boot
RUN set -ex \
 && KERNEL_VERSION=$(rpm -q --queryformat "%{VERSION}-%{RELEASE}.x86_64" kernel) \
 && dracut \
    -f initramfs-${KERNEL_VERSION}.img \
    -i /lib/modules/${KERNEL_VERSION} /lib/modules/${KERNEL_VERSION} \
    ${KERNEL_VERSION}
