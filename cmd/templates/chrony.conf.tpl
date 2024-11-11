# Welcome to the chrony configuration file. See chrony.conf(5) for more
# information about usable directives.

# In case no custom NTP server is provided
# Cloudflare offers a free public time service that allows us to use their
# anycast network of 180+ locations to synchronize time from their closest server.
# See https://blog.cloudflare.com/secure-time/

{{- range .NTPServers}}
pool {{ .Address }} iburst
{{- end }}

# This directive specify the location of the file containing ID/key pairs for
# NTP authentication.
keyfile /etc/chrony/chrony.keys

# This directive specify the file into which chronyd will store the rate
# information.
driftfile /var/lib/chrony/chrony.drift

# Uncomment the following line to turn logging on.
#log tracking measurements statistics

# Log files location.
logdir /var/log/chrony

# Stop bad estimates upsetting machine clock.
maxupdateskew 100.0

# This directive enables kernel synchronisation (every 11 minutes) of the
# real-time clock. Note that it can’t be used along with the 'rtcfile' directive.
rtcsync

# Step the system clock instead of slewing it if the adjustment is larger than
# one second, but only in the first three clock updates.
makestep 1 3