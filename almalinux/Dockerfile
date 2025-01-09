ARG BASE_OS_VERSION

FROM golang:1.22-bullseye AS ignition-builder
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

FROM almalinux:${BASE_OS_VERSION}
# Beware, ARGs for ENV variables except FROM line must be below FROM

ARG GOLLDPD_VERSION
ARG FRR_VERSION

COPY context/etc/initramfs-tools/conf.d/ /etc/initramfs-tools/conf.d/
COPY --from=ignition-builder /work/ignition/bin/amd64/ignition* /usr/local/bin/

RUN set -ex \
 # with an empty /etc/machine-id nothing will be installed in /boot
 # machine-id must a valid, see man machine-id
 && echo "7b9f54e12f3b4f06a34c9d0f4b0e68c2" > /etc/machine-id \
 && dnf update -y \
 && dnf install -y epel-release \
 && dnf update -y \
 && dnf install -y \
    audit \
    ca-certificates \
    chrony \
    cloud-init \
    dbus \
    efibootmgr \
    gettext \
    gnupg2 \
    grub2 \
    grub2-efi-x64 \
    grub2-efi-x64-modules \
    # see https://access.redhat.com/articles/3166931 for details
    grubby \
    haveged \
    ipmitool \
    jq \
    kernel \
    kernel-core \
    logrotate \
    lvm2 \
    mdadm \
    openssh-clients \
    openssh-server \
    openssl \
    os-prober \
    passwd \
    procps \
    rsyslog \
    shim \
    shim-x64 \
    socat \
    sssd \
    sudo \
    systemd \
    systemd-networkd \
    vim \
    watchdog \
    wget \
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
 && curl -fLsS https://rpm.frrouting.org/repo/${FRR_VERSION}-repo-1-0.el9.noarch.rpm -o /tmp/frr-repo.rpm \
 && dnf install -y /tmp/frr-repo.rpm \
 && dnf install -y frr frr-pythontools

COPY context/ /
# Set default target to multi-user.target - otherwise it will be graphical.target
# Fix WantedBy= definition of frr.service - network-online
# is a target that is not pulled in per default during startup
RUN systemctl set-default multi-user.target \
 && systemctl enable lldpd.service \
                     frr.service \
                     watchdog.service \
                     serial-getty@ttyS1.service \
 && systemctl disable serial-getty@tty1.service \
 && systemctl mask serial-getty@tty1.service \
 && systemctl unmask getty.target \
 && systemctl unmask systemd-remount-fs.service \
 && update-ca-trust

# execute dracut to regenerate initramfs with all drivers because otherwise only
# a few drivers are included and the kexec into almalinux will fail because no disks were found
WORKDIR /boot
RUN set -ex \
 && KERNEL_VERSION=$(rpm -q --queryformat "%{VERSION}-%{RELEASE}.x86_64" kernel) \
 && dracut \
    -f initramfs-${KERNEL_VERSION}.img \
    --fscks "umount mount /sbin/fsck* e2fsck" \
    -i /lib/modules/${KERNEL_VERSION} /lib/modules/${KERNEL_VERSION} \
    ${KERNEL_VERSION}
