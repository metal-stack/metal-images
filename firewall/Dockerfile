ARG BASE_OS_NAME
ARG BASE_OS_VERSION
ARG DROPTAILER_VERSION=v0.2.14
ARG FIREWALL_CONTROLLER_VERSION=v2.3.0
ARG TAILSCALE_VERSION=v1.50.1

FROM ghcr.io/metal-stack/droptailer-client:${DROPTAILER_VERSION} AS droptailer-artifacts

FROM ghcr.io/metal-stack/firewall-controller:${FIREWALL_CONTROLLER_VERSION} as firewall-controller-artifacts

FROM ghcr.io/tailscale/tailscale:${TAILSCALE_VERSION} AS tailscale-artifacts

FROM ${BASE_OS_NAME}:${BASE_OS_VERSION}

ENV DEBCONF_NONINTERACTIVE_SEEN="true" \
    DEBIAN_FRONTEND="noninteractive" \
    NODE_EXPORTER_VERSION=1.8.0 \
    NFTABLES_EXPORTER_VERSION=v0.2.9

# iptables reinstallation is required, so that image works in mini-lab environment.
# Most likely the reason is that old config is removed during deletion.
RUN apt-get update --quiet \
 && apt-get install --yes \
    bridge-utils \
    fever \
    tcpdump \
 && apt-get remove --yes iptables \
 && apt install --yes iptables

# Context:
# - chrony.conf provides default configuration for chrony
# - nftables rules are copied in place; nftables.service will load them on startup.
# - systemd.generator in `/etc/systemd/system-generators` is used to run applications in custom VRF
# - systemd.network configuration in `/etc/systemd/network` is used to apply network interface renaming in a bullet-proofed way
COPY context/ /

# Install suricata IDS
RUN /suricata-and-chrony-installation.sh

# Install frr from distribution
# also remove the most recent version of frr and install the distribution version
# this should bring back frr-8.1 on ubuntu and frr-7.5.1 on debian
# TODO can be reverted once https://github.com/FRRouting/frr/issues/12391 is fixed upstream
# This issue is fixed with 8.5.3 but we still face redundancy issues.
# must be done after suricata installation because we need to remove the
# apt sources entries, otherwise we will get back v8.5.x of frr.
# TODO update to frr from 24.04
COPY --from=ghcr.io/metal-stack/frr:9.1-ubuntu-22.04 /artifacts/*.deb /tmp/

RUN set -ex \
 && rm -f /etc/apt/sources.list.d/* \
 && apt-get remove --yes frr frr-pythontools --option=Dpkg::Options::=--force-confdef \
 && apt install --yes /tmp/frr*.deb --no-install-recommends  --option=Dpkg::Options::=--force-confdef \
 && apt --yes autoremove

RUN systemctl disable chrony \
 && systemctl mask chrony \
 && systemctl disable systemd-time-wait-sync.service \
 && systemctl mask systemd-time-wait-sync.service \
 && chmod +x /etc/systemd/system-generators/systemd-vrf-generator \
 && mkdir /etc/vrf \
 && echo chrony > /etc/vrf/systemd.conf \
 && mv /lib/systemd/system/chrony.service /lib/systemd/system/chrony@.service \
 && systemctl enable nftables \
 && systemctl enable fever \
 && systemctl enable frr.service

# Fix permissions of systemd service files
RUN chmod 0644 /lib/systemd/system/*.service

# Install the firewall-controller that might be enabled to connect to a k8s api-server to configure the firewall based on CRDs
COPY --from=firewall-controller-artifacts /firewall-controller /usr/local/bin/firewall-controller

# Install droptailer-client
COPY --from=droptailer-artifacts /client /usr/local/bin/droptailer-client

# Install tailscaled and Tailscale client
COPY --from=tailscale-artifacts /usr/local/bin/tailscaled /usr/local/bin/tailscaled
COPY --from=tailscale-artifacts /usr/local/bin/tailscale /usr/local/bin/tailscale

# Install node_exporter
WORKDIR /tmp/node_exporter
RUN curl -fLsS https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz -o node_exporter.tgz \
 && tar xf node_exporter.tgz --strip-components=1 \
 && mv node_exporter /usr/local/bin/ \
 && rm -rf /tmp/*

# Install nftables-exporter
WORKDIR /tmp/
RUN curl -fLsS https://github.com/metal-stack/nftables-exporter/releases/download/${NFTABLES_EXPORTER_VERSION}/nftables-exporter.tgz -o nftables-exporter.tgz \
 && tar xvf nftables-exporter.tgz -C / \
 && systemctl enable nftables-exporter \
 && rm -rf /tmp/*

# Remove getty overrides to enable console password again
RUN rm -rf /etc/systemd/system/getty@.service.d /etc/systemd/system/serial-getty@.service.d