#!/bin/bash

export MACHINE_TYPE=$1
export ROUTER_ID=10.1.0.1
export ASN=4200003073

if hash goss 2>/dev/null; then
    echo "goss is already installed"
else
    echo "installing goss"
    curl -L https://github.com/aelsabbahy/goss/releases/latest/download/goss-linux-amd64 -o /usr/local/bin/goss
    chmod +rx /usr/local/bin/goss
fi

goss validate -f documentation --color --retry-timeout 30s --sleep 1s