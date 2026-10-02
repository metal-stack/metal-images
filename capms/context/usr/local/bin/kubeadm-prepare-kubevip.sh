#!/usr/bin/env bash

# Path to the frr.conf
FRR_CONF="/etc/frr/frr.conf"

# Extract ASN from the config file using grep and awk
ASN=$(grep -oP "^router bgp \K[0-9]+$" $FRR_CONF | head -n 1)

# Check if ASN was extracted successfully
if [ -z "$ASN" ]; then
    echo "Error: ASN could not be extracted from $FRR_CONF"
    exit 1
fi

# Loop over all files in /etc/kubernetes/manifests
for file in /etc/kubernetes/manifests/*; do
    if [[ -f "$file" ]]; then
        # Replace METAL_MACHINE_ASN in the current file
        sed -i "s/METAL_MACHINE_ASN/$ASN/g" "$file"
        echo "Replaced METAL_MACHINE_ASN with $ASN in $file"
    fi
done
