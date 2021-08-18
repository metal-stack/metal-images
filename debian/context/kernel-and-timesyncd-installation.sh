#!/usr/bin/env bash
set -e
source /etc/os-release

ADDITIONAL_PACKAGES="openssh-server systemd-timesyncd intel-microcode"

if [ "${ID}" = "ubuntu" ] ; then
    echo "Ubuntu - Install kernel, openssh-server and systemd-timesyncd from ubuntu repository"
    apt-get install --yes "linux-generic-hwe-${SEMVER_MAJOR_MINOR}" ${ADDITIONAL_PACKAGES}
else
    echo "Debian - Install kernel, openssh-server and systemd-timesyncd from backports repository"
    # Note: for firewall images the backports kernel is a hard requirements because kernel >= 5.x is necessary for vxlan/evpn
    # we need openssh-server because of
    #
    # openssh (1:8.1p1-5) unstable; urgency=medium
    # * Apply upstream patches to allow clock_nanosleep() and variants in the
    #   seccomp sandbox, fixing failures with glibc 2.31
    # s. https://metadata.ftp-master.debian.org/changelogs/main/o/openssh/testing_changelog
    #
    # with ssh to the test vm one gets an audit event for the clock_nanosleep syscall (230)
    # audit: type=1326 audit(1595317960.526:2): auid=4294967295 uid=107 gid=65534 ses=4294967295 subj=kernel pid=1177 comm="sshd" exe="/usr/sbin/sshd" sig=31 arch=c000003e syscall=230 compat=0 ip=0x7fc40d5eebea code=0x0
    echo "deb https://deb.debian.org/debian ${VERSION_CODENAME} contrib" > /etc/apt/sources.list.d/contrib.list
    echo "deb https://deb.debian.org/debian ${VERSION_CODENAME}-backports main contrib non-free" > /etc/apt/sources.list.d/backports.list
    apt-get update --quiet
    apt-get install --yes -t buster-backports ${ADDITIONAL_PACKAGES}
    echo "deb https://deb.debian.org/debian testing main" > /etc/apt/sources.list.d/testing.list
    apt-get update --quiet
    apt-get install --yes -t testing linux-image-amd64
    # remove testing list, otherwise doing update on the machine will show 100s of missing updates.
    rm -f /etc/apt/sources.list.d/testing.list
fi

# Remove WIFI, netronome, v4l and liquidio firmware to save ~300MB image size
rm -rf /usr/lib/firmware/*wifi* \
    /usr/lib/firmware/netronome \
    /usr/lib/firmware/v4l* \
    /usr/lib/firmware/liquidio \
    /var/lib/apt/lists/*
