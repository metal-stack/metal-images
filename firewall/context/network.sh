#!/usr/bin/env bash
cd /etc/metal/networker || exit
./metal-networker firewall configure --input "${INSTALL_YAML}"