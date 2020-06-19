ARG BASE_OS_NAME
ARG BASE_OS_VERSION
ARG DROPTAILER_VERSION=v0.2.2
ARG FIREWALL_CONTROLLER_VERSION=v0.1.2
ARG NFTABLES_EXPORTER_BRANCH=eb592159

FROM metalstack/droptailer-client:${DROPTAILER_VERSION} AS droptailer-artifacts

FROM metalstack/firewall-controller:${FIREWALL_CONTROLLER_VERSION} as firewall-controller-artifacts

FROM golang:1.14 AS nftables_exporter_builder
RUN git clone https://github.com/Sheridan/nftables_exporter.git \
 && cd nftables_exporter \
 && git checkout ${NFTABLES_EXPORTER_BRANCH} \
 && make \
 && strip ./bin/nftables_exporter \
 && mv ./bin/nftables_exporter /

FROM ${BASE_OS_NAME}:${BASE_OS_VERSION}

ENV DEBCONF_NONINTERACTIVE_SEEN="true" \
    DEBIAN_FRONTEND="noninteractive" \
    VERSION_NODE_EXPORTER=0.18.1 \
    EVEBOX_VERSION=1:0.11.1

RUN apt-get update --quiet \
 && apt-get install --yes \
    bridge-utils \
    chrony \
    nftables \
    tcpdump \
 && apt-get remove --yes iptables

# Install suricate event forwarding agent evebox
RUN curl -q https://evebox.org/files/GPG-KEY-evebox | apt-key add - \
 && echo "deb http://evebox.org/files/debian stable main" | tee /etc/apt/sources.list.d/evebox.list \
 && apt-get update --quiet \
 && apt-get install evebox=${EVEBOX_VERSION} \
 # remove provided systemd units, they get created by metal-networker with the appropriate vrf set.
 && rm -f /lib/systemd/system/evebox-agent.service /lib/systemd/system/evebox.service

# Context:
# - chrony.conf provides default configuration for chrony
# - ifupdown2 has a bug that disables forwarding for ipv4 after an ifup.
#   This can be solved by adding a policy to `/var/lib/ifupdown2/policy.d/` that requires forwarding.
#   Source: Discussion with Attilla (cumulus) @ https://cumulusnetworks.slack.com/archives/GGE01MNPN/p1554790876006800
# - nftables rules are copied in place; nftables.service will load them on startup.
# - systemd.generator in `/etc/systemd/system-generators` is used to run applications in custom VRF
# - systemd.network configuration in `/etc/systemd/network` is used to apply network interface renaming in a bullet-proofed way
COPY context/ /

# Install suricata IDS
RUN /suricata-installation.sh

# Pre-Configure chrony instead of systemd-timesyncd because it is able to run in a VRF context without issues.
# Final setup is left to metal-networker that knows the internet-facing VRF.
# To succeed metal-networker enabling chrony it is important to provide the chrony unit template in advance.
# Usually the generator creates that template but the generator is loaded only after system boot or at `systemctl daemon-reload` (cannot be run from Docker Context).
RUN systemctl disable systemd-timesyncd \
 && systemctl disable chronyd \
 && systemctl mask chronyd \
 && chmod +x /etc/systemd/system-generators/systemd-vrf-generator \
 && mkdir /etc/vrf \
 && echo chrony > /etc/vrf/systemd.conf \
 && mv /lib/systemd/system/chrony.service /lib/systemd/system/chrony@.service \
 && systemctl enable nftables

# Fix permissions of systemd service files
RUN chmod 0644 /lib/systemd/system/*.service

# Install the firewall-controller that might be enabled to connect to a k8s api-server to configure the firewall based on CRDs
COPY --from=firewall-controller-artifacts /firewall-controller /usr/local/bin/firewall-controller

# Install droptailer-client
COPY --from=droptailer-artifacts /client /usr/local/bin/droptailer-client

# Install nftables_exporter
COPY --from=nftables_exporter_builder /nftables_exporter /usr/local/bin/

# Install node_exporter
WORKDIR /tmp/node_exporter
RUN curl -fLsS https://github.com/prometheus/node_exporter/releases/download/v${VERSION_NODE_EXPORTER}/node_exporter-${VERSION_NODE_EXPORTER}.linux-amd64.tar.gz -o node_exporter.tgz \
 && tar xf node_exporter.tgz --strip-components=1 \
 && mv node_exporter /usr/local/bin/ \
 && rm -rf /tmp/*
