#!/usr/bin/env bash

set -e

TIMEOUT=${TIMEOUT:-900}

echo "wait for the machine test, it is run by goss.service in the VM"
rc=""
for _ in $(seq "${TIMEOUT}"); do
  rc=$(sed -n 's/.*GOSS_EXIT_CODE=\([0-9]*\).*/\1/p' ./console.log | head -n1)
  if [ -n "${rc}" ]; then
    break
  fi
  sleep 1
done

echo "console output of the VM"
cat ./console.log

echo "Terminating cloud-hypervisor processes"
killall cloud-hypervisor
rm -f ./my.sock

if [ -z "${rc}" ]; then
  echo "goss tests did not finish within ${TIMEOUT}s"
  exit 1
fi
echo "goss tests finished with exit code ${rc}"
exit "${rc}"
