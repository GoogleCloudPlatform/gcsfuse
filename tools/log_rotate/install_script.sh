#!/usr/bin/env bash

# Create directory, which keeps all logrotate-config to rotate hourly.
mkdir -p /etc/logrotate.hourly.d

# By default, only daily cron executes the logrotate which rotates the logs
# according to the configuration in /etc/logrotate.d folder. To support the
# hourly execution of logrotate, we have created logrotate config-files specific
# hourly-execution.

# This file describes about the hourly execution of logrotate. All the logrotate
# -config files present in /etc/logrotate.hourly.d will be executed hourly.
cat << EOF | tee /etc/logrotate.hourly.conf
# This to enforce logrotate to run as root/adm. In ubuntu distributions, the
# ownership of the file created via rsyslog is syslog hence, we need to add
# this explicitly.
su root adm

# packages drop hourly log rotation information into this directory
include /etc/logrotate.hourly.d
EOF
chmod 644 /etc/logrotate.hourly.conf

# Add a shell script which will be run hourly, which eventually executes the
# command to rotate the logs according to config present in /etc/logrotate.hourly.d
cat << EOF | tee /etc/cron.hourly/gcsfuse-logrotate
#!/bin/bash
test -x /usr/sbin/logrotate || exit 0
/usr/sbin/logrotate /etc/logrotate.hourly.conf
EOF

# Make sure, we have hourly logrotate setup inplace correctly.
if [ $? -eq 0 ]; then
        echo "Hourly cron setup for logrotate completed successfully"
else
        echo "Please install linux package - cron"
        exit 1
fi

chmod 775 /etc/cron.hourly/gcsfuse-logrotate

# Logrotate configuration to rotate gcsfuse logs.
cat << EOF | tee /etc/logrotate.hourly.d/gcsfuse
/var/log/gcsfuse.log {
  rotate 10
  size 5G
  missingok
  notifempty
  compress
  dateext
  dateformat -%Y%m%d-%s
  delaycompress
  sharedscripts
  postrotate
	  systemctl kill -s HUP rsyslog.service >/dev/null 2>&1 || true
  endscript
}
EOF

# Make sure gcsfuse-logrotate config got placed correctly.
if [ $? -eq 0 ]; then
        echo "Logrotate config for gcsfuse updated successfully!"
else
        echo "Please install linux package - logrotate"
        exit 1
fi

# Syslog configuration to filter and redirect the logs from /var/log/syslog to
# /var/log/gcsfuse.log. The prefix-number 08 in the gcsfuse.conf is just for
# file ordering and precedence. Like, if the same parameter configuration is
# present in 10-x.conf and 20-y.conf the latter will overwrite first one.
cat > /etc/rsyslog.d/08-gcsfuse.conf <<EOF

# Change the ownership of create log file through rsyslog.
\$umask 0000
\$FileCreateMode 0644

# Redirect all "gcsfuse" logs to /var/log/gcsfuse.log
:programname, isequal, "gcsfuse" {
  *.* /var/log/gcsfuse.log
  stop
}
EOF

# Make sure gcsfuse-syslog filter config got placed correctly.
if [ $? -eq 0 ]; then
        echo "Syslog config for gcsfuse updated successfully!"
else
        echo "Please install linux package - rsyslog"
        exit 1
fi

# Restart the syslog service after adding the syslog configuration.
systemctl restart rsyslog
