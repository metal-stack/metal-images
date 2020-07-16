#!/bin/bash

set -e

# example: sudo ./start.sh quay.io/metalstack/ubuntu:19.10 

if [ ! hash footloose 2>/dev/null || ! hash ignite 2>/dev/null ]; then
  echo "please install footloose and ignite"
fi

export IMAGE="${1}"

echo "import oci to ignite: ${IMAGE}"
ignite image rm -f ${IMAGE} || true
ignite image import --runtime=docker ${IMAGE}

echo "generate footloose config"
envsubst < footloose.yaml.tpl > footloose.yaml

echo "creat ignite / firecracker vm with footloose"
footloose create

echo "determine ip address of vm"
export IP=$(ignite inspect vm cluster-images0 -t "{{ .Status.IPAddresses }}")

while ! nc -z ${IP} 22; do
  echo "ssh is not available yet"
  sleep 2
done

echo "ssh is available"

./test.sh