#!/usr/bin/env bash

# Create the log file for gcsfuse package.
mkdir -p /var/log/gcsfuse/

# Installs logrotate configuration files.
mkdir -p /etc/logrotate.d/

cat > /etc/logrotate.d/gcsfuse <<EOF
/var/log/gcsfuse/*.log {
    rotate ${LOGROTATE_FILES_MAX_COUNT:-5}
    copytruncate
    missingok
    notifempty
    compress
    maxsize ${LOGROTATE_MAX_SIZE:-100M}
    daily
    dateext
    dateformat -%Y%m%d-%s
    create 0644 root root
}
EOF
