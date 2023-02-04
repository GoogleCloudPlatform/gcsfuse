# Playbook for production issues
This page enumerates some common user facing issues around GCSFuse and also discusses potential solutions to the same.

## Installation
### GCSFuse fails with Docker container
Though not tested extensively, the [community](https://stackoverflow.com/questions/65715624/permission-denied-with-gcsfuse-in-unprivileged-ubuntu-based-docker-container) reports that GCSFuse works only in privileged mode when used with Docker. There are [solutions](https://github.com/samos123/gke-gcs-fuse-unprivileged) which exist and claim to do so without privileged mode, but these are not tested by the gcsfuse team.


## Mounting
Most of the common mount point issues are around permissions on both local mount point and the GCS bucket. It is highly recommended to retry with --foreground --debug_fuse --debug_fs --debug_gcs --debug_http flags which would provide much more detailed logs to understand the errors better and possibly provide a solution.

### Mount failed with fusermount exist status 1
This is a very generic error which could mean either the permission to the directory was incorrect ([ref](https://stackoverflow.com/questions/34700393/gcsfuse-mount-exits-with-status-1)) or the permissions to the service account may be insufficient ([ref](https://serverfault.com/questions/911600/while-accessing-fuse-mounted-storage-bucket-its-showing-403-forbidden-error)).

### Mount failed with invalid argument
Please consult the [flags.go](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/flags.go) file to make sure that the flag that you are passing to mount gcsfuse is a valid one.

### Mount successful but files not visible
This is explained in the [semantics](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/semantics.md#implicit-directories) page and can possibly be fixed by passing the --implicit-dirs=true flag to the gcsfuse command.

### Mount failed with fusermount3 exit status 1
The error comes when bucket is already mounted in a folder and we try to mount it again in the same folder without un-mounting it. Fix it to first umount and then remount. Command for un-mounting - 
sudo umount mounted-dir-name

## Serving
Once the mounting is successful, there are other issues which may crop up during the serving phase and this section discusses some of those and their possible remedies.

### Input/Output Error
This issue is also related to permissions and most likely the culprit is the bucket not having the right permissions for gcsfuse to operate upon ([ref](https://stackoverflow.com/questions/36382704/gcsfuse-input-output-error)).

### Generic NO_PUBKEY Error - while installing gcsfuse on ubuntu 22.04

While running

*   sudo apt-get update

Err:18 http://dl.google.com/linux/chrome/deb stable Release.gpg \
The following signatures couldn't be verified because the public key is not
available: NO_PUBKEY 6494C6D6997C215E

Enter NO_PUBKEY here (eg: 6494C6D6997C215E is NO_PUBKEY for above error)

**Solution**

*   sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 6494C6D6997C215E
*   sudo apt-get update