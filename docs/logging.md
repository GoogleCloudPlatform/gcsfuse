# GCSFuse User Logging Guide
This readme describes about the logging support in gcsfuse and how users can use
that support.

GCSFuse supports different debug flags e.g. --debug_fs, --debug_fuse, 
--debug_gcs etc. Each of these flags logs debug info of a different component
of gcsfuse. For more information, use `gcsfuse --help`.

## Log location
GCSFuse logs its activity to a file if the user specifies one with the `--log-file`
flag. Otherwise, it logs to **stdout** in the foreground and to **syslog** in the background.

## Log rotation
GCSFuse does not automatically rotate logs, so you must configure this manually.

### To support the rotation of logs written to syslog
1. Please make sure you have `rsyslog`, `logrotate` and `cron` packages installed
and running on the system.
E.g. commands to install and run the packages on ubuntu/deb:
```bash
sudo add-apt-repository ppa:adiscon/v8-devel
sudo apt-get update
sudo apt-get install rsyslog logrotate cron

sudo systemctl start rsyslog
sudo systemctl start logrotate
sudo systemctl start cron
```
2. Run the shell script located at [this](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/tools/log_rotate/install_script.sh) location.
3. Mount the GCS bucket using GCSFuse command with logs enabled in background mode.
Also, verify the logs present at `/var/log/gcsfuse.log`.
4. Run the below command to test this setup.
```bash
# The below command should run successfully, it will trigger the hourly cron job.
# Cron will trigger logrotate and you will find the logrotate status in
# /var/log/logrotate/status file.
sudo run-parts -v /etc/cron.hourly 
```

### To support the rotation of logs written to custom log-file
E.g. assume you pass `--log-file=/$HOME/test.log`, gcsfuse will start writing
the logs to `/$HOME/test.log`. Now, to support log rotation, you can follow the
below steps:

1. Create a logrotate configuration file (gcsfuse-custom-logrotate-conf):
```bash
/$HOME/test.log {
    rotate 10
    size 5G
    missingok
    notifempty
    compress
    dateext
    dateformat -%Y%m%d-%s
    copytruncate
}
```

Here,
``` text
* rotate 10 - means only 10 backup files will be kept other than original file.
* size 5G - rotation will take place only when file size exceeds this.
* missingok - if a log file is missing, logrotate will not issue an error.
* notifempty - do not rotate the log if log files are empty.
* compress - old versions of log files are compressed with gzip by default.
* dateext - archive logs with date extension instead of adding a number.
* dateformat %Y%m%d-%s - added %s to make a unique rotate-log file name in hourly cron setup.
* copytruncate - truncate the original log file in place after creating a copy, instead 
of moving the old log file and optionally creating a new one.
```
2. You can verify the above created config using the below logrotate command:
   `/usr/sbin/logrotate -d <config_file_path>`
3. Create a cron job which will execute the logrotate command periodically. E.g.
the below command creates a cron job which runs every 2 minute. 
`echo "*/2 * * * * /usr/sbin/logrotate <abs_path_to_config_file> --state <abs_path_to_state_file> | crontab _`  
Note: `--state` flag is useful if logrotate is being run as different user, it saves
status of logrotate execution as we get by default in `/var/lib/logrotate/status`.

