#!/usr/bin/env bash
set -e

CA_FILES="FI-TS_F04_Root_CA_G1.der FI-TS_F04_Class1_CA_G2.der FI-TS_F04_Class2_CA_G2.der"

cd /tmp
for CA_FILE in $CA_FILES;
do
    echo "installing ${CA_FILE}"
    curl -fLOsS http://pki.f-i-ts.de/ca-certs-F04_G1/${CA_FILE}
    dest=$(basename ${CA_FILE})
    dest="${dest}.crt"
    openssl x509 -in ${CA_FILE} -inform der -outform pem -out ${dest}
    mv ${dest} /usr/local/share/ca-certificates/
    rm ${CA_FILE}
done