# the debian image has no binutils, strip in a throwaway stage which still has apt
FROM baseapp AS strip
RUN set -ex \
 && apt-get update \
 && apt-get install --yes --no-install-recommends binutils \
 # crictl is the only binary shipped with debug symbols
 && strip /usr/local/bin/crictl

FROM baseapp

# Remove everything which is not needed on a kubernetes node, especially the package management (apt, dpkg)
# and perl. Nodes are never updated in place, they get reinstalled with a new image instead.
RUN set -ex \
 && REMOVE_PKGS="apt apt-utils libapt-pkg.* dpkg debconf debconf-i18n libdebconfclient0 install-info tasksel.* \
                 perl perl-base perl-modules-.* libperl.* libtext-charwidth-perl libtext-wrapi18n-perl \
                 debian-archive-keyring gpgv sqv ucf sensible-utils runit-helper linux-base adduser \
                 dhcpcd-base \
                 openssh-server openssh-client openssh-sftp-server libfido2-1 libcbor0.10 \
                 curl libcurl4t64 libnghttp2-14 libnghttp3-9 libpsl5t64 librtmp1 libssh2-1t64 libbrotli1 \
                 libfreetype6 libpng16-16t64 \
                 libgnutls30t64 libp11-kit0 libnettle8t64 libhogweed6t64 libtasn1-6 libldap2 libsasl2-2 libsasl2-modules-db \
                 wget apparmor-utils python3-apparmor python3-libapparmor media-types gettext-base eject" \
 # regular expressions, no globbing while joining them
 && PKG_RE="$(set -f; echo ${REMOVE_PKGS} | tr ' ' '|')" \
 && REMOVE_RE="^(${PKG_RE})\$" \
 # inventory of the remaining packages, used by export.sh
 && dpkg-query -W -f='${Package} ${Version}\n' | awk -v re="${REMOVE_RE}" '$1 !~ re' > /etc/metal/packages.txt \
 # package-owned files have to be resolved before dpkg is gone
 && for p in $(dpkg-query -W -f='${Package}\n' | grep -E "${REMOVE_RE}"); do dpkg -L "${p}"; done | sort -u > /tmp/remove.list \
 # prune the status file to the remaining packages, it is kept outside of /var/lib/dpkg for the SBOM
 && awk -v re="^Package: (${PKG_RE})\$" 'BEGIN { RS=""; ORS="\n\n" } { split($0, l, "\n"); if (l[1] !~ re) print }' \
      /var/lib/dpkg/status > /tmp/status \
 # mkinitramfs asks dpkg for the architecture, it is called by os-installer on RAID setups
 && sed -i 's/^DPKG_ARCH=.*/DPKG_ARCH=amd64/' /usr/sbin/mkinitramfs \
 # use the linux-version replacement from the context for kernel version comparison
 && sed -i 's/dpkg --compare-versions/linux-version compare/' /usr/share/initramfs-tools/hooks/intel_microcode \
 # delete files and symlinks only, directories may be shared with other packages
 && while read -r f; do if [ -L "${f}" ] || [ -f "${f}" ]; then rm -f "${f}"; fi; done < /tmp/remove.list \
 # remove remaining perl scripts which can not run anymore (update-rc.d, deb-systemd-helper, ...)
 && find /usr/bin /usr/sbin /usr/lib /usr/share -type f -perm /111 \
      -exec sh -c 'for f; do head -n1 "$f" | grep -q "^#!.*perl" && echo "$f"; done; true' _ {} + \
      | xargs -r rm -fv \
 # dangling symlinks, e.g. enabled apt-daily.timer and dpkg-db-backup.timer units, alternatives or sysv init links
 && find /etc/systemd /etc/alternatives /etc/rc?.d /usr/bin /usr/sbin -xtype l -print -delete \
 # privilege separation user of sshd
 && userdel sshd \
 && rm -rf /etc/apt /etc/dpkg /etc/debconf.conf \
 # sshd_config from the debian image, host keys generated in the postinst and the .wants dirs of the ssh units
           /etc/ssh /etc/sv/ssh /etc/systemd/system/ssh*.wants /etc/systemd/system/sshd*.wants \
           /usr/lib/openssh /usr/share/openssh /usr/share/runit/meta/ssh /var/log/runit/ssh /run/sshd \
           /var/lib/ucf/cache/:etc:ssh:sshd_config /var/lib/systemd/deb-systemd-helper-enabled/*ssh* \
           /var/lib/systemd/deb-systemd-helper-enabled/multi-user.target.wants/ssh.service \
           /var/lib/apt /var/cache/apt /var/cache/debconf /var/log/apt /var/log/dpkg.log /var/log/alternatives.log \
           /usr/share/perl* /usr/lib/x86_64-linux-gnu/perl* /usr/share/debconf /usr/share/keyrings \
           /usr/share/doc/* /usr/share/man/* /usr/share/info/* /usr/share/lintian /usr/share/bug /usr/share/menu \
 # locale sources, only used by localedef, C.UTF-8 is built into glibc
           /usr/share/i18n \
 # os-installer only calls ignition
           /usr/local/bin/ignition-validate \
 # the only user of libfreetype6, grub fonts are not generated on the node
           /usr/bin/grub-mkfont \
 # frr daemons which are not enabled in /etc/frr/daemons, zebra, bgpd, staticd and mgmtd are kept
           /usr/lib/frr/babeld /usr/lib/frr/bfdd /usr/lib/frr/eigrpd /usr/lib/frr/fabricd /usr/lib/frr/isisd \
           /usr/lib/frr/ldpd /usr/lib/frr/nhrpd /usr/lib/frr/ospfd /usr/lib/frr/ospf6d /usr/lib/frr/pathd \
           /usr/lib/frr/pbrd /usr/lib/frr/pimd /usr/lib/frr/pim6d /usr/lib/frr/ripd /usr/lib/frr/ripngd /usr/lib/frr/vrrpd \
           /usr/lib/frr/ospfclient.py /usr/lib/frr/frr_babeltrace.py /usr/lib/x86_64-linux-gnu/frr/modules/pathd_pcep.so \
           /usr/bin/mtracebis \
 # python test suite and pydoc topics, bytecode caches are regenerated on demand
           /usr/lib/python3.*/test /usr/lib/python3.*/pydoc_data \
 && find /usr/lib/python3* /usr/share/python3 -name __pycache__ -type d -prune -exec rm -rf {} + \
 && find /usr/share/locale -mindepth 1 -maxdepth 1 ! -name locale.alias -exec rm -rf {} + \
 # kernel modules a bare-metal kubernetes worker never loads
 # drm is kept for the BMC console (ast, mgag200), infiniband for RDMA setups
 && for k in /usr/lib/modules/*; do \
      rm -rf ${k}/kernel/sound \
             ${k}/kernel/drivers/bluetooth \
             ${k}/kernel/drivers/gpu/drm/amd \
             ${k}/kernel/drivers/gpu/drm/i915 \
             ${k}/kernel/drivers/gpu/drm/nouveau \
             ${k}/kernel/drivers/gpu/drm/radeon \
             ${k}/kernel/drivers/gpu/drm/xe \
             ${k}/kernel/drivers/iio \
             ${k}/kernel/drivers/input/joystick \
             ${k}/kernel/drivers/media \
             ${k}/kernel/drivers/net/wireless \
             ${k}/kernel/drivers/staging \
             ${k}/kernel/net/bluetooth \
             ${k}/kernel/net/mac80211 \
             ${k}/kernel/net/wireless; \
      depmod "$(basename ${k})"; \
    done \
 # keep the dpkg status and the checksums of the files which still exist for the SBOM,
 # syft finds dpkg databases by the glob **/lib/dpkg/status, dpkg itself would not use this location
 && SBOM_DB=/usr/share/metal/sbom/lib/dpkg \
 && mkdir -p "${SBOM_DB}/info" \
 && cp /tmp/status "${SBOM_DB}/status" \
 && for p in $(awk '/^Package: /{ print $2 }' /tmp/status); do \
      for m in "/var/lib/dpkg/info/${p}.md5sums" "/var/lib/dpkg/info/${p}:amd64.md5sums"; do \
        [ -f "${m}" ] || continue; \
        while read -r sum f; do \
          if [ -e "/${f}" ] || [ -L "/${f}" ]; then echo "${sum}  ${f}"; fi; \
        done < "${m}" > "${SBOM_DB}/info/$(basename "${m}")"; \
      done; \
    done \
 && rm -rf /var/lib/dpkg /tmp/* \
 # fail if one of the removed tools is still present
 && for b in apt apt-get dpkg dpkg-query perl ssh sshd curl grub-mkfont; do \
      if command -v "${b}"; then echo "${b} is still present"; exit 1; fi; \
    done

# copied after the cleanup, otherwise the dpkg-reconfigure shim would be removed together with debconf
COPY --from=ctx . /
COPY --from=strip /usr/local/bin/crictl /usr/local/bin/crictl
