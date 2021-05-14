#!/usr/bin/env bash
set -e
cd /etc/metal/networker
./metal-networker machine configure --input "${INSTALL_YAML}"