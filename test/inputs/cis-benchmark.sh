#!/usr/bin/env bash
# Somehow /etc/resolve.conf is created with read permissions granted only to root.
# As apt drops privileges during download this won't work. Allow reading for all.
chmod 644 /etc/resolv.conf

# Install and configure OVH-CIS benchmark
apt update && apt install -y git
rm /var/log/apt/*
git clone -b ${CIS_VERSION} --depth 1 https://github.com/ovh/debian-cis.git
cd debian-cis
cp debian/default /etc/default/cis-hardening
sed -i "s#CIS_ROOT_DIR=.*#CIS_ROOT_DIR='$(pwd)'#" /etc/default/cis-hardening

# Disable inapropriate checks
bin/hardening.sh --create-config-files-only --allow-unsupported-distribution --batch

disable-testcase () {
  CONFFILE="etc/conf.d/$1.cfg"
  if [ -f "$CONFFILE" ]; then
    sed --i -E "s/^(status=).+/\1disabled/g" "$CONFFILE"
  else
    echo "status=disabled" > "$CONFFILE"
  fi
}

grep -o '^[^#]*' /cis-disabled.txt | while read testcases; do
  for i in bin/hardening/$testcases ; do
    disable-testcase `basename $i .sh`
  done
done

# Run benchmark
bin/hardening.sh --audit --allow-unsupported-distribution --batch | grep -v ^OK
