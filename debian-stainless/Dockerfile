FROM baseapp AS stripped

# include this stage in the SBOM attestation, it still carries a (pruned) dpkg status file
ARG BUILDKIT_SBOM_SCAN_STAGE=true

# Remove everything which is not needed on a kubernetes node, especially the package management (apt, dpkg)
# and perl. Nodes are never updated in place, they get reinstalled with a new image instead.
RUN set -ex \
 && REMOVE_PKGS="apt apt-utils libapt-pkg.* dpkg debconf debconf-i18n libdebconfclient0 install-info tasksel.* \
                 perl perl-base perl-modules-.* libperl.* libtext-charwidth-perl libtext-wrapi18n-perl \
                 debian-archive-keyring gpgv sqv ucf sensible-utils runit-helper linux-base adduser \
                 dhcpcd-base \
                 wget apparmor-utils python3-apparmor python3-libapparmor media-types gettext-base eject" \
 # regular expressions, no globbing while joining them
 && PKG_RE="$(set -f; echo ${REMOVE_PKGS} | tr ' ' '|')" \
 && REMOVE_RE="^(${PKG_RE})\$" \
 # inventory of the remaining packages, used by export.sh
 && dpkg-query -W -f='${Package} ${Version}\n' | awk -v re="${REMOVE_RE}" '$1 !~ re' > /etc/metal/packages.txt \
 # package-owned files have to be resolved before dpkg is gone
 && for p in $(dpkg-query -W -f='${Package}\n' | grep -E "${REMOVE_RE}"); do dpkg -L "${p}"; done | sort -u > /tmp/remove.list \
 # prune the status file to the remaining packages, it is only kept for the SBOM scan of this stage
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
 # dangling symlinks, e.g. enabled apt-daily.timer and dpkg-db-backup.timer units or alternatives
 && find /etc/systemd /etc/alternatives /usr/bin /usr/sbin -xtype l -print -delete \
 && find /var/lib/dpkg -mindepth 1 -delete \
 && cp /tmp/status /var/lib/dpkg/status \
 && rm -rf /etc/apt /etc/dpkg /etc/debconf.conf \
           /var/lib/apt /var/cache/apt /var/cache/debconf /var/log/apt /var/log/dpkg.log /var/log/alternatives.log \
           /usr/share/perl* /usr/lib/x86_64-linux-gnu/perl* /usr/share/debconf /usr/share/keyrings \
           /usr/share/doc/* /usr/share/man/* /usr/share/info/* /usr/share/lintian /usr/share/bug /usr/share/menu \
 && find /usr/share/locale -mindepth 1 -maxdepth 1 ! -name locale.alias -exec rm -rf {} + \
 # kernel modules a bare-metal kubernetes worker never loads
 # gpu/drm is kept for the BMC console, infiniband for RDMA setups
 && for k in /usr/lib/modules/*; do \
      rm -rf ${k}/kernel/sound \
             ${k}/kernel/drivers/bluetooth \
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
 && rm -rf /tmp/* \
 # fail if one of the removed tools is still present
 && for b in apt apt-get dpkg dpkg-query perl; do \
      if command -v "${b}"; then echo "${b} is still present"; exit 1; fi; \
    done

# copied after the cleanup, otherwise the dpkg-reconfigure shim would be removed together with debconf
COPY --from=ctx . /

FROM stripped

# the package database was only kept for the SBOM scan of the previous stage
RUN rm -rf /var/lib/dpkg
