#!/usr/bin/env bash
if [ -e "/etc/metal/userdata" ]; then
    firstLine=$(sed '1q;d' /etc/metal/userdata)
    secondLine=$(sed '2q;d' /etc/metal/userdata)
    if [[ ${firstLine} == "#cloud-config" ]] || [[ "${secondLine}" == "#cloud-config" ]]; then
        echo "run cloud-init with userdata"
        cloud-init --file /etc/metal/userdata -d single --name write_files
        cloud-init --file /etc/metal/userdata -d single --name runcmd
        /var/lib/cloud/instances/iid-datasource-none/scripts/runcmd
    fi
fi