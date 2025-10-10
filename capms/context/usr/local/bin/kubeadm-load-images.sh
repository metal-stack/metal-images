#!/usr/bin/env bash

for TAR in /kubeadm-images/*.tar; do
  ctr -n k8s.io images import $TAR
done
