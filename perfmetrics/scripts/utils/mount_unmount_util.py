# Copyright 2024 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import subprocess

def unmount_gcs_bucket(gcs_bucket,log) -> None:
  """Unmounts the GCS bucket.

  Args:
    gcs_bucket: Name of the directory to which GCS bucket is mounted to.

  Raises:
    An error if the GCS bucket could not be umnounted and aborts the program.
  """

  log.info('Unmounting the GCS Bucket.\n')
  exit_code = subprocess.call('umount -l {}'.format(gcs_bucket), shell=True)
  if exit_code != 0:
    log.error('Error encountered in umounting the bucket. Aborting!\n')
    subprocess.call('bash', shell=True)
  else:
    subprocess.call('rm -rf {}'.format(gcs_bucket), shell=True)
    log.info(
        'Successfully unmounted the bucket and deleted %s directory.\n',
        gcs_bucket)


def mount_gcs_bucket(bucket_name, gcsfuse_flags,log) -> str:
  """Mounts the GCS bucket into the gcs_bucket directory.

  Args:
    bucket_name: Name of the bucket to be mounted.
    gcsfuse_flags: Set of flags for which bucket_name will be mounted

  Returns:
    A string which contains the name of the directory to which the bucket
    is mounted.

  Raises:
    Aborts the program if error is encountered while mounting the bucket.
  """

  log.info('Started mounting the GCS Bucket using GCSFuse.\n')
  gcs_bucket = bucket_name
  log.info(f'{bucket_name} {gcs_bucket} {gcsfuse_flags}')
  subprocess.call('mkdir {}'.format(gcs_bucket), shell=True)

  exit_code = subprocess.call(
      'gcsfuse {} {} {}'.format(
          gcsfuse_flags, bucket_name, gcs_bucket), shell=True)
  if exit_code != 0:
    log.error('Cannot mount the GCS bucket due to exit code %s.\n', exit_code)
    subprocess.call('bash', shell=True)
  else:
    return gcs_bucket