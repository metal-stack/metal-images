ARG BASE_OS_NAME
ARG BASE_OS_VERSION
ARG DROPTAILER_VERSION=v0.2.10
ARG FIREWALL_CONTROLLER_VERSION=v1.1.5

FROM ghcr.io/metal-stack/droptailer-client:${DROPTAILER_VERSION} AS droptailer-artifacts

FROM ghcr.io/metal-stack/firewall-controller:${FIREWALL_CONTROLLER_VERSION} as firewall-controller-artifacts

FROM ${BASE_OS_NAME}:${BASE_OS_VERSION}

ENV DEBCONF_NONINTERACTIVE_SEEN="true" \
    DEBIAN_FRONTEND="noninteractive" \
    NODE_EXPORTER_VERSION=1.3.1 \
    NFTABLES_EXPORTER_VERSION=v0.1.5

RUN apt-get update --quiet \
 && apt-get install --yes --no-install-recommends \
    bridge-utils \
    fever \
    tcpdump \
 && apt-get remove --yes iptables \
 && apt --yes autoremove

# Context:
# - chrony.conf provides default configuration for chrony
# - nftables rules are copied in place; nftables.service will load them on startup.
# - systemd.generator in `/etc/systemd/system-generators` is used to run applications in custom VRF
# - systemd.network configuration in `/etc/systemd/network` is used to apply network interface renaming in a bullet-proofed way
COPY context/ /

# Install suricata IDS
RUN /suricata-and-chrony-installation.sh

# Pre-Configure chrony instead of systemd-timesyncd because it is able to run in a VRF context without issues.
# Final setup is left to metal-networker that knows the internet-facing VRF.
# To succeed metal-networker enabling chrony it is important to provide the chrony unit template in advance.
# Usually the generator creates that template but the generator is loaded only after system boot or at `systemctl daemon-reload` (cannot be run from Docker Context).
# systemd-time-wait-sync.service is disabled because it sometimes does not start and blocks depending services like logrotate.
# see https://github.com/systemd/systemd/issues/14061
RUN systemctl disable systemd-timesyncd \
 && systemctl disable chrony \
 && systemctl mask chrony \
 && systemctl disable systemd-time-wait-sync.service \
 && systemctl mask systemd-time-wait-sync.service \
 && chmod +x /etc/systemd/system-generators/systemd-vrf-generator \
 && mkdir /etc/vrf \
 && echo chrony > /etc/vrf/systemd.conf \
 && mv /lib/systemd/system/chrony.service /lib/systemd/system/chrony@.service \
 && systemctl enable nftables \
 && systemctl enable fever

# Fix permissions of systemd service files
RUN chmod 0644 /lib/systemd/system/*.service

# Install the firewall-controller that might be enabled to connect to a k8s api-server to configure the firewall based on CRDs
COPY --from=firewall-controller-artifacts /firewall-controller /usr/local/bin/firewall-controller

# Install droptailer-client
COPY --from=droptailer-artifacts /client /usr/local/bin/droptailer-client

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
 # TODO remove once metal-networker adopted systemd service template to new binary location
 && ln -s /usr/bin/nftables-exporter /usr/local/bin/nftables_exporter \
 && rm -rf /tmp/*

# Remove getty overrides to enable console password again
RUN rm -rf /etc/systemd/system/getty@.service.d /etc/systemd/system/serial-getty@.service.d