"""Contains tasks which reads data from files using tensorflow's tf.data api.

Contains tasks for reading tfrecord files of sizes: 100MiB & 200MiB using
GCSFuse mounted directory and from GCS using tensorflow's internal GCS client.

Example:

  lg_obj = load_generator.LoadGenerator(...)
  task_obj = tf_data.TFDataReadGCSFuseAutotune100MB()
  observations = lg_obj.generate_load(task_obj)
"""
import os
import logging
import tensorflow as tf
from load_generator import task

# .tfrecord file contains multiple TFRecords in it. This defines the size of
# that single TFRecord in bytes.
SINGLE_RECORD_SIZE = 256 * 1024


def _create_tfrecord_file(file_path, file_size):
  """Creates .tfrecord file of given file size in bytes at given file path.

  Doesn't create a file if file is already present.
  """
  # We only check existence in this case because actual TFRecord's file size is
  # not exactly equal to file_size
  if tf.io.gfile.exists(file_path):
    return

  logging.info('Creating TFRecord file %s of size %s.', file_path, file_size)
  content = b'\t' * SINGLE_RECORD_SIZE
  writer = tf.io.TFRecordWriter(file_path)
  for _ in range(0, file_size, len(content)):
    writer.write(content)
  writer.close()


class TFDataRead(task.LoadTestTask):
  """Base task class for reading tfrecord file using tensorflow's tf.data api.

  Note: (a) This class is abstract because it doesn't define task method.
  (b) It is mandatory to define task method, keep TASK_TYPE = 'read' in base
  class and to assign appropriate values of FILE_PATH_FORMAT, FILE_SIZE,
  PREFETCH, NUM_PARALLEL_CALLS, NUM_FILES in derived classes.
  """
  TASK_TYPE = 'read'
  FILE_PATH_FORMAT = ''
  FILE_SIZE = 0
  PREFETCH = 0
  NUM_PARALLEL_CALLS = 0
  NUM_FILES = 0
  SHARD = 1

  def __init__(self):
    super().__init__()
    self.file_names = [
        self.FILE_PATH_FORMAT.format(file_num=file_num)
        for file_num in range(self.NUM_FILES)
    ]

  def _tf_data_task(self, file_names):
    """Reads TFRecord files using tensorflow's tf.data method.

    Note: It is important to return length of content read because the returned
    value is used by post task of load generator to compute avg. bandwidth.

    Args:
      file_names: List of String paths to files.

    Returns:
      Integer denoting the size of content read in bytes.
    """
    content_len = 0
    files_dataset = tf.data.Dataset.from_tensor_slices(file_names)

    def tfrecord(path):
      return tf.data.TFRecordDataset(path).prefetch(self.PREFETCH).shard(
          self.SHARD, 0)

    dataset = files_dataset.interleave(
        tfrecord,
        cycle_length=self.NUM_PARALLEL_CALLS,
        num_parallel_calls=self.NUM_PARALLEL_CALLS)
    for record in dataset:
      content_len = content_len + len(record.numpy())

    return content_len

  def create_files(self, num_processes):  #pylint: disable=unused-argument
    """Creates self.NUM_FILES number of files to be used in load testing.
    """
    logging.info('Creating %s bytes TFRecord files using the format '
                 '%s', self.FILE_SIZE, self.FILE_PATH_FORMAT)

    file_path_0 = self.FILE_PATH_FORMAT.format(file_num=0)
    _create_tfrecord_file(file_path_0, self.FILE_SIZE)
    for file_num in range(1, self.NUM_FILES):
      file_path = self.FILE_PATH_FORMAT.format(file_num=file_num)
      if not tf.io.gfile.exists(file_path):
        logging.info('Creating TFRecord file %s of size %s.', file_path,
                     self.FILE_SIZE)
        tf.io.gfile.copy(file_path_0, file_path)

  def pre_task(self, assigned_process_id, assigned_thread_id):
    """Pre-task to clear OS's page caches in load testing.
    """
    # Clear the page cache as there is no way to bypass cache in tf.data.
    # Note: This takes time and hence decrease the average bandwidth.
    os.system("sudo sh -c 'sync; echo 3 >  /proc/sys/vm/drop_caches'")


# Disabling doc string pylint beyond this because it is not expected to write
# doc string for each task class when it is available for base class.
#pylint: disable=missing-class-docstring
class TFDataReadGCSFuseAutotune100MB(TFDataRead):

  TASK_NAME = 'GCSFUSE_100MB_AUTOTUNE'

  FILE_PATH_FORMAT = '/gcs/100mb/read.{file_num}.tfrecord'
  FILE_SIZE = 100 * 1024 * 1024
  PREFETCH = -1
  NUM_PARALLEL_CALLS = -1
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadGCSFuse16P100MB(TFDataRead):

  TASK_NAME = 'GCSFUSE_100MB_16P'

  FILE_PATH_FORMAT = '/gcs/100mb/read.{file_num}.tfrecord'
  FILE_SIZE = 100 * 1024 * 1024
  PREFETCH = 100
  NUM_PARALLEL_CALLS = 16
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadGCSFuse64P100MB(TFDataRead):

  TASK_NAME = 'GCSFUSE_100MB_64P'

  FILE_PATH_FORMAT = '/gcs/100mb/read.{file_num}.tfrecord'
  FILE_SIZE = 100 * 1024 * 1024
  PREFETCH = 100
  NUM_PARALLEL_CALLS = 64
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadGCSFuse100P100MB(TFDataRead):

  TASK_NAME = 'GCSFUSE_100MB_100P'

  FILE_PATH_FORMAT = '/gcs/100mb/read.{file_num}.tfrecord'
  FILE_SIZE = 100 * 1024 * 1024
  PREFETCH = 100
  NUM_PARALLEL_CALLS = 100
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadGCSFuseShard100P100MB(TFDataRead):

  TASK_NAME = 'GCSFUSE_SHARD_100MB_100P'

  FILE_PATH_FORMAT = '/gcs/100mb/read.{file_num}.tfrecord'
  FILE_SIZE = 100 * 1024 * 1024
  PREFETCH = 100
  NUM_PARALLEL_CALLS = 100
  NUM_FILES = 1024
  SHARD = 800

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadAutotune100MB(TFDataRead):

  TASK_NAME = 'TF_100MB_AUTOTUNE'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/100mb/' \
                     'read.{file_num}.tfrecord'
  FILE_SIZE = 100 * 1024 * 1024
  PREFETCH = -1
  NUM_PARALLEL_CALLS = -1
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataRead16P100MB(TFDataRead):

  TASK_NAME = 'TF_100MB_16P'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/100mb/' \
                     'read.{file_num}.tfrecord'
  FILE_SIZE = 100 * 1024 * 1024
  PREFETCH = 100
  NUM_PARALLEL_CALLS = 16
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataRead64P100MB(TFDataRead):

  TASK_NAME = 'TF_100MB_64P'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/100mb/' \
                     'read.{file_num}.tfrecord'
  FILE_SIZE = 100 * 1024 * 1024
  PREFETCH = 100
  NUM_PARALLEL_CALLS = 64
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataRead100P100MB(TFDataRead):

  TASK_NAME = 'TF_100MB_100P'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/100mb/' \
                     'read.{file_num}.tfrecord'
  FILE_SIZE = 100 * 1024 * 1024
  PREFETCH = 100
  NUM_PARALLEL_CALLS = 100
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadShard100P100MB(TFDataRead):

  TASK_NAME = 'TF_SHARD_100MB_100P'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/100mb/' \
                     'read.{file_num}.tfrecord'
  FILE_SIZE = 100 * 1024 * 1024
  PREFETCH = 100
  NUM_PARALLEL_CALLS = 100
  NUM_FILES = 1024
  SHARD = 800

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadGCSFuseAutotune200MB(TFDataRead):

  TASK_NAME = 'GCSFUSE_200MB_AUTOTUNE'

  FILE_PATH_FORMAT = '/gcs/200mb/read.{file_num}.tfrecord'
  FILE_SIZE = 200 * 1024 * 1024
  PREFETCH = -1
  NUM_PARALLEL_CALLS = -1
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadGCSFuse16P200MB(TFDataRead):

  TASK_NAME = 'GCSFUSE_200MB_16P'

  FILE_PATH_FORMAT = '/gcs/200mb/read.{file_num}.tfrecord'
  FILE_SIZE = 200 * 1024 * 1024
  PREFETCH = 200
  NUM_PARALLEL_CALLS = 16
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadGCSFuse64P200MB(TFDataRead):

  TASK_NAME = 'GCSFUSE_200MB_64P'

  FILE_PATH_FORMAT = '/gcs/200mb/read.{file_num}.tfrecord'
  FILE_SIZE = 200 * 1024 * 1024
  PREFETCH = 200
  NUM_PARALLEL_CALLS = 64
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadGCSFuse100P200MB(TFDataRead):

  TASK_NAME = 'GCSFUSE_200MB_100P'

  FILE_PATH_FORMAT = '/gcs/200mb/read.{file_num}.tfrecord'
  FILE_SIZE = 200 * 1024 * 1024
  PREFETCH = 200
  NUM_PARALLEL_CALLS = 100
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadGCSFuseShard100P200MB(TFDataRead):

  TASK_NAME = 'GCSFUSE_SHARD_200MB_100P'

  FILE_PATH_FORMAT = '/gcs/200mb/read.{file_num}.tfrecord'
  FILE_SIZE = 200 * 1024 * 1024
  PREFETCH = 200
  NUM_PARALLEL_CALLS = 100
  NUM_FILES = 1024
  SHARD = 800

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadAutotune200MB(TFDataRead):

  TASK_NAME = 'TF_200MB_AUTOTUNE'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/200mb/' \
                     'read.{file_num}.tfrecord'
  FILE_SIZE = 200 * 1024 * 1024
  PREFETCH = -1
  NUM_PARALLEL_CALLS = -1
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataRead16P200MB(TFDataRead):

  TASK_NAME = 'TF_200MB_16P'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/200mb/' \
                     'read.{file_num}.tfrecord'
  FILE_SIZE = 200 * 1024 * 1024
  PREFETCH = 200
  NUM_PARALLEL_CALLS = 16
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataRead64P200MB(TFDataRead):

  TASK_NAME = 'TF_200MB_64P'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/200mb/' \
                     'read.{file_num}.tfrecord'
  FILE_SIZE = 200 * 1024 * 1024
  PREFETCH = 200
  NUM_PARALLEL_CALLS = 64
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataRead100P200MB(TFDataRead):

  TASK_NAME = 'TF_200MB_100P'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/200mb/' \
                     'read.{file_num}.tfrecord'
  FILE_SIZE = 200 * 1024 * 1024
  PREFETCH = 200
  NUM_PARALLEL_CALLS = 100
  NUM_FILES = 1024

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)


class TFDataReadShard100P200MB(TFDataRead):

  TASK_NAME = 'TF_SHARD_200MB_100P'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/200mb/' \
                     'read.{file_num}.tfrecord'
  FILE_SIZE = 200 * 1024 * 1024
  PREFETCH = 200
  NUM_PARALLEL_CALLS = 100
  NUM_FILES = 1024
  SHARD = 800

  def task(self, assigned_process_id, assigned_thread_id):
    return self._tf_data_task(self.file_names)
