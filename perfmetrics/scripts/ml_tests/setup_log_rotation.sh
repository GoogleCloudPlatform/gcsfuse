#!/bin/bash

# Add a shell script which will be run hourly, which eventually executes the
# command to rotate the logs according to config present in /etc/logrotate.hourly.d
cat << EOF | tee /etc/cron.hourly/gcsfuse_logrotate
#!/bin/bash
test -x /usr/sbin/logrotate || exit 0
/usr/sbin/logrotate ${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/gcsfuse_logrotate.conf --state ${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/gcsfuse_logrotate_status"
EOF

# Make sure, we have hourly logrotate setup inplace correctly.
if [ $? -eq 0 ]; then
        echo "Hourly cron setup for logrotate completed successfully"
else
        echo "Please install linux package - cron"
        exit 1
fi

chmod 775 /etc/cron.hourly/gcsfuse-logrotate