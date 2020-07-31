#!/usr/bin/env bash
cd /etc/metal/networker || exit
./metal-networker firewall configure --input /etc/metal/install.yaml