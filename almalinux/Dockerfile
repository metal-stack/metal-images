ARG BASE_OS_VERSION

FROM golang:1.22-bullseye as ignition-builder
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

FROM almalinux:${BASE_OS_VERSION}
# Beware, ARGs for ENV variables except FROM line must be below FROM

ARG GOLLDPD_VERSION
ARG FRR_VERSION

COPY context/etc/initramfs-tools/conf.d/ /etc/initramfs-tools/conf.d/
COPY --from=ignition-builder /work/ignition/bin/amd64/ignition* /usr/local/bin/

RUN set -ex \
 # with an empty /etc/machine-id nothing will be installed in /boot
 && echo "almalinux" > /etc/machine-id \ 
 && dnf install -y \
    ca-certificates \
    chrony \
    cloud-init \
    curl \
    dbus \
    efibootmgr \
    gettext \
    gnupg2 \
    grub2 \
    grub2-efi-x64 \
    grub2-efi-x64-modules \
    ipmitool \
    jq \
    kernel \
    kernel-core \
    logrotate \
    lvm2 \
    mdadm \
    openssl \
    openssh-clients \
    openssh-server \
    passwd \
    procps \
    shim \
    shim-x64 \
    socat \
    sudo \
    systemd \
    watchdog \
    wget \
    vim \
    yum-utils
RUN curl -fLsS https://github.com/metal-stack/go-lldpd/releases/download/${GOLLDPD_VERSION}/go-lldpd.tgz -o /tmp/go-lldpd.tgz \
 && tar -xf /tmp/go-lldpd.tgz \
 && echo "LANG=\"en_US.UTF-8\"" > /etc/default/locale \
 && cp /usr/share/zoneinfo/Europe/Berlin /etc/localtime \
 && rm -rf /tmp/* \
 # create directory to store metal-networker tmp files
 && mkdir -p /etc/metal/networker \
 # clean yum cache
 && yum clean all

# Install FRR see https://rpm.frrouting.org/ for available channels
RUN set -ex \
 && curl -fLsS https://rpm.frrouting.org/repo/${FRR_VERSION}-repo-1-0.el8.noarch.rpm -o /tmp/frr-repo.rpm \
 && dnf install -y /tmp/frr-repo.rpm \
 && dnf install -y frr frr-pythontools

RUN echo "---" >> /etc/metal/build-meta.yaml \
 && echo "ignitionVersion: ${IGNITION_BRANCH}" >> /etc/metal/build-meta.yaml \
 && echo "golldpdVersion: ${GOLLDPD_VERSION}" >> /etc/metal/build-meta.yaml \
 && echo "frrVersion: ${FRR_VERSION}" >> /etc/metal/build-meta.yaml

COPY context/ /
# Set default target to multi-user.target - otherwise it will be graphical.target
# Fix WantedBy= definition of frr.service - network-online
# is a target that is not pulled in per default during startup
# FIXME systemctl enable can be remove if the preset mechanism works
# in /usr/lib/systemd/system-preset/metal.preset
RUN systemctl set-default multi-user.target \
 && systemctl enable lldpd.service \
                     frr.service \
                     watchdog.service \
 && update-ca-trust

# execute dracut to regenerate initramfs with all drivers because otherwise only
# a few drivers are included and the kexec into centos will fail because no disks where found
WORKDIR /boot
RUN set -ex \
 && KERNEL_VERSION=$(rpm -q --queryformat "%{VERSION}-%{RELEASE}.x86_64" kernel) \
 && dracut \
    -f initramfs-${KERNEL_VERSION}.img \
    -i /lib/modules/${KERNEL_VERSION} /lib/modules/${KERNEL_VERSION} \
    ${KERNEL_VERSION}
