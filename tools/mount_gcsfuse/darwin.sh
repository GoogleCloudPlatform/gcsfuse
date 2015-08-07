#!/bin/bash

# An "external helper" for mount(8) that can be used to set up compatibility
# with the `mount` command. Edit the variables below if necessary, then install
# as /sbin/mount_gcsfuse.

# Path to the mount_gcsfuse binary.
MOUNT_GCSFUSE=/usr/local/bin/mount_gcsfuse

# Path to a JSON key file downloaded from the Google Developers Console.
#
# This is necessary only if one of the other pieces of the default credentials
# logic (https://goo.gl/VYsojX) doesn't work for your situation. In particular,
# this is not necessary if you have run `gcloud auth login` or are running on a
# Google Cloud Engine VM that was created with the storage-full scope.
KEY_FILE=

# The PATH to use for mount_gcsfuse, which must contain the gcsfuse binary.
WRAPPED_PATH=/usr/local/bin

# Set to an output file where you want stdout and stderr to go, or /dev/null, or
# a syslog "facility.priority" spec. See `man 1 daemon` for more.
OUTPUT=/dev/null

# Run under daemon so that we return to mount(8) immediately.
daemon \
  --env="PATH=$WRAPPED_PATH" \
  --env="GOOGLE_APPLICATION_CREDENTIALS=$KEY_FILE" \
  --output $OUTPUT \
  -- \
  $HELPER "$@"

exit
