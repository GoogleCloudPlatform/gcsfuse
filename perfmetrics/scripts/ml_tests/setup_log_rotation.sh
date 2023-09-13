#!/bin/bash

# This will setup the rotation of log-file present at the $1
# Please provide the absolute path of log-file.

log_file=$1
echo "Creating logrotate configuration..."
cat << EOF | sudo tee $HOME/github/gcsfuse/gcsfuse_logrotate.conf
${log_file} {
  su root adm
  rotate 3
  size 1G
  missingok
  notifempty
  compress
  dateext
  dateformat -%Y%m%d-%s
  copytruncate
}
EOF

# Set the correct access permission to the config file.
sudo chmod 0644 $HOME/github/gcsfuse/gcsfuse_logrotate.conf
sudo chown root $HOME/github/gcsfuse/gcsfuse_logrotate.conf

# Make sure logrotate installed on the system.
if test -x /usr/sbin/logrotate ; then
  echo "Logrotate already installed on the system."
else
  echo "Installing logrotate on the system..."
  sudo apt-get install logrotate
fi

# Add a shell script which will be run hourly, which eventually executes the
# command to rotate the logs according to config present in /etc/logrotate.hourly.d
cat << EOF | sudo tee /etc/cron.hourly/gcsfuse_logrotate
#!/bin/bash
test -x /usr/sbin/logrotate || exit 0
/usr/sbin/logrotate $HOME/github/gcsfuse/gcsfuse_logrotate.conf --state $HOME/github/gcsfuse/gcsfuse_logrotate_status
EOF

# Make sure, we have hourly logrotate setup inplace correctly.
if [ $? -eq 0 ]; then
        echo "Hourly cron setup for logrotate completed successfully"
else
        echo "Please install linux package - cron"
        exit 1
fi

sudo chmod 775 /etc/cron.hourly/gcsfuse_logrotate

# Restart the cron service
sudo service cron restart
