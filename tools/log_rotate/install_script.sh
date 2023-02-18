#!/usr/bin/env bash

# Logrotate configuration to rotate gcsfuse logs.
cat > /etc/logrotate.d/gcsfuse <<EOF
/var/log/gcsfuse/*.log {
  rotate 10
  maxsize 100M
  hourly
  missingok
  notifempty
  compress
  dateext
  delaycompress
  sharedscripts
  postrotate
	  /usr/lib/rsyslog/rsyslog-rotate
  endscript
}
EOF

# Syslog configuration to filter and redirect the logs from /var/log/syslog to
# /var/log/gcsfuse/gcsfuse.log.
cat > /etc/rsyslog.d/12-gcsfuse.conf <<EOF
if $programname == 'gcsfuse' then /var/log/gcsfuse/gcsfuse.log;RSYSLOG_FileFormat
EOF

# Restart the syslog service after adding the syslog configuration.
systemctl restart rsyslog

