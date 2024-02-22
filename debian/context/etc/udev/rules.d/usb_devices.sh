# By default, disable all.
ACTION=="add", SUBSYSTEMS=="usb", TEST=="authorized_default", ATTR{authorized_default}="0"

# Enable hub devices.
ACTION=="add", ATTR{bDeviceClass}=="09", TEST=="authorized", ATTR{authorized}="1"

# Enables keyboard devices
ACTION=="add", ATTR{product}=="*[Kk]eyboard*", TEST=="authorized", ATTR{authorized}="1"

# PS2-USB converter
ACTION=="add", ATTR{product}=="*Thinnet TM*", TEST=="authorized", ATTR{authorized}="1"
