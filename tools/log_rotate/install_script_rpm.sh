#!/usr/bin/env bash

# Create directory, which keeps all logrotate-config to rotate hourly.
mkdir /etc/logrotate.hourly.d

cat << EOF | tee /etc/logrotate.hourly.conf
# packages drop hourly log rotation information into this directory
include /etc/logrotate.hourly.d
EOF

chmod 644 /etc/logrotate.hourly.conf

cat << EOF | tee /etc/cron.hourly/logrotate
#!/bin/bash
test -x /usr/sbin/logrotate || exit 0
/usr/sbin/logrotate /etc/logrotate.hourly.conf
EOF

chmod 775 /etc/cron.hourly/logrotate

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
	  kill -HUP \$(cat /var/run/rsyslogd.pid) >/dev/null 2>&1 || true
  endscript
}
EOF

chmod 644 /etc/logrotate.hourly.d/gcsfuse

# Make sure gcsfuse-logrotate config got placed correctly.
if [ $? -eq 0 ]; then
        echo "Logrotate config for gcsfuse updated successfully!"
else
        echo "Please install linux package - logrotate"
        exit 1
fi

# Syslog configuration to filter and redirect the logs from /var/log/syslog to
# /var/log/gcsfuse.log.
cat > /etc/rsyslog.d/08-gcsfuse.conf <<EOF

# Change the ownership of create log file through rsyslog.
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
