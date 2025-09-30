ARG BASE_OS_NAME=ghcr.io/metal-stack/ubuntu
ARG BASE_OS_VERSION=24.04

FROM baseapp AS kubeadm-base

ARG KUBE_APT_BRANCH
ARG KUBE_VERSION
ARG KUBE_VIP_VERSION

ENV DEBCONF_NONINTERACTIVE_SEEN="true" \
    DEBIAN_FRONTEND="noninteractive"

COPY --from=ctx ./ /

RUN curl -fsSL https://pkgs.k8s.io/core:/stable:/${KUBE_APT_BRANCH}/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg \
 && echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/${KUBE_APT_BRANCH}/deb/ /" > /etc/apt/sources.list.d/kubernetes.list \
 && apt update \
 && apt install --yes --no-install-recommends \
        kubeadm=${KUBE_VERSION}-1.1 \
        kubectl=${KUBE_VERSION}-1.1 \
        kubelet=${KUBE_VERSION}-1.1 \
        kubernetes-cni

FROM kubeadm-base AS kubeadm-pull

ARG CRANE_RELEASE=https://github.com/google/go-containerregistry/releases/download/latest/go-containerregistry_Linux_x86_64.tar.gz
ARG CRANE_CHECKSUM

ADD --checksum=${CRANE_CHECKSUM} "${CRANE_RELEASE}" /download/gocontainerregistry.tar
RUN tar -xzf /download/gocontainerregistry.tar \
 && chmod +x /crane \
 && mkdir /kubeadm-images \
 && bash -c 'for img in $(kubeadm config images list --kubernetes-version ${KUBE_VERSION}); do \
        /crane pull --format tarball $img /kubeadm-images/$(basename $img).tar; \
    done' \
 && /crane pull --format tarball ghcr.io/kube-vip/kube-vip:${KUBE_VIP_VERSION} /kubeadm-images/kube-vip.tar

FROM kubeadm-base AS kubeadm-runtime

COPY --from=kubeadm-pull /kubeadm-images /kubeadm-images
