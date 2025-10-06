

ARG BASE_IMAGE
FROM ${BASE_IMAGE} AS base

ARG MACHINE_TYPE=machine
ARG GOSS_VERSION=v0.4.7
ARG GOSS_URL=https://github.com/goss-org/goss/releases/download/${GOSS_VERSION}/goss-linux-amd64

COPY ./ctx /

RUN rm /etc/systemd/system/getty.target.wants/getty@tty1.service \
 && mv /etc/metal/${MACHINE_TYPE}.yaml /etc/metal/install.yaml \
 && mv /etc/metal/userdata-${MACHINE_TYPE}.json /etc/metal/userdata

ADD --chmod=755 "${GOSS_URL}" /usr/local/bin/goss