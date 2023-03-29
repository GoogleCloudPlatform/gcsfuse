# Prerequisites

1. Before invoking Cloud Storage FUSE, you must have a Cloud Storage bucket that you want to mount. If you haven't yet, [create](https://cloud.google.com/storage/docs/creating-buckets#storage-create-bucket-console) a storage bucket. 
2. Provide [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials#howtheywork) to authenticate Cloud Storage FUSE requests to Cloud Storage. By default, Cloud Storage FUSE automatically loads the Application Default Credentials without any further configuration if they exist. You can use the gcloud auth login command to easily generate Application Default Credentials.

        gcloud auth application-default login
        gcloud auth login
        gcloud auth list

   Alternatively, you can authenticate Cloud Storage FUSE by setting the --key-file flag to the path of a JSON key file, which you can download from the Google Cloud console. You can also set the GOOGLE_APPLICATION_CREDENTIALS environment variable to the path of the JSON key.

        GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json gcsfuse [...]
When mounting with an fstab entry, use the key_file option:

        my-bucket /mount/point gcsfuse rw,noauto,user,key_file=/path/to/key.json

When you create a Compute Engine VM, its service account can also be used to authenticate access to Cloud Storage FUSE. When Cloud Storage FUSE is run from such a VM, it automatically has access to buckets owned by the same project as the VM.

# Basic syntax for mounting

The base syntax for using Cloud Storage FUSE is:

    gcsfuse [global options] [bucket] mountpoint

Where [global options] are optional specific flags you can pass (use gcsfuse --help), [bucket] is the optional name of your bucket, and ‘mountpoint’ is the directory on your machine that you are mounting the bucket to. For example:
    
    gcsfuse my-bucket /usr/me/myself

# Static Mounting

Static mounting means mounting a specific bucket. For example, say I want to mount the bucket “my-bucket” to the directory /usr/me/myself

    mkdir /usr/me/myself
    gcsfuse my-bucket /usr/me/myself
Note: Avoid using the name of the bucket as the local directory mount point name.

