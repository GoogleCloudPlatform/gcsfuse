# GCSFuse User Logging Guide
Here, we will describe about the logging in gcsfuse and how different users
can use the logging support in GCSFuse.

GCSFuse supports different debug_flags named, --debug_fs, --debug_fuse, 
--debug_gcs etc. Each of these flags logs debug info of a different components
of gcsfuse.

## Log Location
GCSFuse supports a flag named `--log-file` to control the location of the logs.
By default, the GCSFuse writes the logs to syslog (see syslog section below for
more details) but user can pass custom location (including stdout and stderr)
using '--log-file' flag.

The below sections describe how users can use default and custom locations
support for logs along with how log rotation can be achieved in those cases.

### Default location - syslog
By default, if you don't provide --log-file, GCSFuse writes all the logs to
syslog which eventually redirects to `/var/log/gcsfuse.log`. The redirection happens
with the help of rsyslog conf file. The rsyslog conf file is present at
`/etc/rsyslog.d/08-gcsfuse.conf`, if GCSFuse is installed using released-package.
Otherwise, the configuration has to be added manually. E.g., in case when you
mount by building the source code.

#### Log rotation
We achieve the rotation of logs with the help of logrotate package/config, which
is applied on the log file (/var/log/gcsfuse.log). You can find the logrotate
configuration placed at `/etc/logrotate.hourly.d/gcsfuse` in case of installation
using released package. Otherwise, you need to add the logrotate configuration
manually to support the log-rotation.

If you want to mount by building gcsfuse from code. You need to execute
[this script](https://github.com/add_after_merge) with root permission.
This script will apply all the required configuration inplace which happens
during installation when we install gcsfuse using release packages.

### Custom log location
You can specify any custom location by passing the location path to `--log-file`

Note: log rotation is not supported by default in this case (unlike the case of
default location i.e. syslog)

#### Write to stdout/err
We can find the file descriptor corresponding to stdout/err and pass in --log-file
flag to redirect the log to stdout/err.

We have equivalent file path for stdout/err. You can use:
`/dev/fd/1` -> `stdout`
`/dev/fd/2` -> `stderr`

Or you may try finding the equivalent path using the below command:

Command to find: `ls -la /dev | grep 'std'`
Output would be: `stderr -> /proc/self/fd/2`
`stdin  -> /proc/self/fd/0`
`stdout -> /proc/self/fd/1`

Hence, to redirect the logs to stdout we can pass `--log-file=/proc/self/fd/1` as
an argument of gcsfuse command.

#### Log rotation
To support the log rotation, you need to create the log-rotation config manually
and apply on the passed log-location.

E.g., let's assume you pass `--log-file=/$HOME/test.log`, gcsfuse will start writing
the logs to `/$HOME/test.log`. Now, to support log rotation, you can follow the
below steps:

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
    copytruncate
}
```

Here,
``` text
rotate 10 - means only 10 backup files will be kept other than original file.
size 5G - rotation will take place only when file size exceeds this.
missingok - if a log file is missing, logrotate will not issue an error.
notifempty - do not rotate the log if log files are empty.
compress - old versions of log files are compressed with gzip by default.
dateext - archive logs with date extension instead of adding a number.
dateformat %Y%m%d-%s - added %s to make a unique rotate-log file name in hourly cron setup.
```
2. Add the above created files to `/etc/logrotate.hourly.d/`. This directory is created 
during the installation of gcsfuse to rotate the logs hourly. In case, this directory
doesn't exist, or you want to rotate the logs daily, you can put the above created
config file to `/etc/logrotate.d/`.
3. You can verify the config using logrotate command:
`/usr/sbin/logrotate -d <config_file_path>`
