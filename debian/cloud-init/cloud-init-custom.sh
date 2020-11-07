#!/usr/bin/env bash
if [ -e "/etc/metal/userdata" ]; then
    cloud-init --file /etc/metal/userdata -d single --name write_files
    cloud-init --file /etc/metal/userdata -d single --name runcmd
    /var/lib/cloud/instances/iid-datasource-none/scripts/runcmd
fi