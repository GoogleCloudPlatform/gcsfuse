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

### Custom log location
When you specify --log-file for the custom log location, you will find all the
logs in the custom file only. This is the same behavior as before the syslog
implementation.

To support the log rotation, you need to create the log-rotation config manually
to apply on the passed log-location.

Steps to support log-rotation in case of custom log-file:
TODO: Add steps to configure log-rotation for custom log-file.

### Log location - stdout/err
We can find the file descriptor corresponding to stdout/err and pass in --log-file
flag to redirect the log to stdout/err.

Command to find: `ls -la /dev | grep 'std'`
Output would be: `stderr -> /proc/self/fd/2`
                 `stdin  -> /proc/self/fd/0`
                 `stdout -> /proc/self/fd/1`

Hence, to redirect the logs to stdout we can use `--log-file=/proc/self/fd/1`.



