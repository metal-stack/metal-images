FROM quay.io/metalstack/ubuntu:20.04 AS vmlinux-builder

RUN apt-get update && apt-get install --yes binutils wget zstd
RUN wget -O metal-kernel https://github.com/metal-stack/kernel/releases/download/5.10.52-67/metal-kernel
RUN wget -O extract-vmlinux https://raw.githubusercontent.com/torvalds/linux/master/scripts/extract-vmlinux
RUN chmod +x extract-vmlinux
RUN rm -rf /boot/* && rm -rf /lib/modules/*
RUN ./extract-vmlinux metal-kernel > /boot/vmlinux
RUN ls -alh /boot

FROM scratch
COPY --from=vmlinux-builder /boot/ /boot/
COPY --from=vmlinux-builder /lib/modules /lib/modules/
