#!/usr/bin/env bash
# Logrotate configuration to rotate gcsfuse logs.
cat > /etc/logrotate.d/gcsfuse <<EOF
/var/log/gcsfuse.log {
  rotate 10
  size 5G
  hourly
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

# Make sure gcsfuse-logrotate config got placed correctly.
if [ $? -eq 0 ]; then
        echo "Logrotate config for gcsfuse updated successfully!"
else
        echo "Please install linux package - logrotate"
        exit 1
fi

# Syslog configuration to filter and redirect the logs from /var/log/syslog to
# /var/log/gcsfuse.log.
cat > /etc/rsyslog.d/12-gcsfuse.conf <<EOF

# Change the ownership of create log file through rsyslog.
\$umask 0000
\$FileCreateMode 0644

if \$programname  == 'gcsfuse' then /var/log/gcsfuse.log;RSYSLOG_FileFormat
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
