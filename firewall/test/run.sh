#!/bin/bash

ignite rm -f firewall || true
ignite run metalstack/images/firewall:2.0-ubuntu \
        --ssh \
        --debug \
        --name firewall \
        -f $PWD/sshd_config:/etc/ssh/sshd_config \
        -f $PWD/sudoers:/etc/sudoers \
        -f $PWD/resolv.conf:/etc/resolv.conf
