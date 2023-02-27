# GCSFuse User Logging Guide
Here, we will describe about the logging in gcsfuse and how different types of user
can use the control to complete their work.

GCSFuse supports different types of debug_flag named, --debug_fs, --debug_fuse, 
--debug_gcs etc., which logs the debug info of different component in gcsfuse.

## Log Location
GCSFuse provides a flag named --log-file which controls the location of the logs.
Either it can be to syslog or custom location. We will achieve the logging at
stdout or stderr with the help of same flag.

We will also describe how we can achieve the rotation of logs in each case.

### Default location - syslog
By default, if you don't provide --log-file, GCSFuse writes all the logs to
syslog which eventually redirects to `/var/log/gcsfuse.log`. The redirection happens
with the help of rsyslog conf file, which you found at /etc/rsyslog.d/08-gcsfuse.conf
folder if you have installed the gcsfuse using the released-package, otherwise
you need add this configuration to make this work.

We achieve the rotation of logs with the help of logrotate config applying on
the log file (/var/log/gcsfuse.log). You can find the logrotate configuration
placed at `/etc/logrotate.hourly.d/gcsfuse` in case of installation using released
package. Otherwise, you need to add the logrotate configuration manually to support
the log-rotation.

If you want to mount by building gcsfuse from code. You need to execute
[this script](https://github.com/add_after_merge) with root permission.
This script will apply all the required configuration inplace which happens
during installation when we install gcsfuse using release packages.

### Custom log location
When you specify --log-file for the custom log location, you will find all the
logs in the custom file only. This is the same behavior as before the syslog
implementation.

To support the log rotation, you need to create the log-rotation config manually
to apply on the passed log-location.

Let's assume you pass `--log-file=/$HOME/test.log`, gcsfuse will start writing
the debug information on this file. To support log-rotation for this file, you can
follow the below steps:
1. Create a logrotate configuration file named with any name like, gcsfuse-custom-logrotate-conf:
```bash
/$HOME/test.log {
    rotate 10
    size 5G
    missingok
    notifempty
    compress
    dateext
    dateformat -%Y%m%d-%s
    delaycompress
    copytruncate
}
```
2. Add the above created files to `/etc/logrotate.hourly.d/` this directory is created 
during the installation of gcsfuse to rotate the logs hourly. In case if this directory
doesn't exist, or you want to rotate daily not hourly, you can put the above created
config file to `/etc/logrotate.d/`.
3. You can verify the config using logrotate command:
`/usr/sbin/logrotate -d <config_file_path>` 

### Log location - stdout/err
We can find the file descriptor corresponding to stdout/err and pass in --log-file
flag to redirect the log to stdout/err.

Command to find: `ls -la /dev | grep 'std'`
Output would be: `stderr -> /proc/self/fd/2`
                 `stdin  -> /proc/self/fd/0`
                 `stdout -> /proc/self/fd/1`

Hence, to redirect the logs to stdout we can pass `--log-file=/proc/self/fd/1` as
an argument of gcsfuse command.



