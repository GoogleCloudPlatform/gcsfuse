"""Contains tasks that read data from files using tensorflow's tf.io.gfile api.

Contains tasks for reading file sizes: 256KiB, 3MiB, 5MiB, 50MiB, 100MiB,
200MiB. TFGFileRead is the base class for all the tasks.

Example:

  lg_obj = load_generator.LoadGenerator(...)
  task_obj = python_os.TFGFileRead256KB()
  observations = lg_obj.generate_load(task_obj)
"""
import logging
import tensorflow as tf
from load_generator import task


def _create_binary_file(file_path, file_size):
  """Creates binary file of given file size in bytes at given GCS path.

  Doesn't create a file if file is already present and has same file size.
  """
  if tf.io.gfile.exists(file_path) and tf.io.gfile.stat(
      file_path).length == file_size:
    return
  logging.info('Creating file %s of size %s.', file_path, file_size)
  with tf.io.gfile.GFile(file_path, 'wb') as f_p:
    content = b'\t' * file_size
    f_p.write(content)


class TFGFileRead(task.LoadTestTask):
  """Base task class for reading GCS file using tensorflow's tf.io.gfile api.

  Note: (a) This class is abstract because it doesn't define task method.
  (b) It is mandatory to define task method, keep TASK_TYPE = 'read' and to
  assign appropriate values of FILE_PATH_FORMAT, FILE_SIZE and BLOCK_SIZE in
  derived classes.
  """
  TASK_TYPE = 'read'
  FILE_PATH_FORMAT = ''
  FILE_SIZE = 0
  BLOCK_SIZE = 0

  def _tf_read_task(self, file_path, file_size, block_size):
    """Reads file of given size from given GCS file path and block size.

    Note: It is important to return length of content read because the returned
    value is used by post task of load generator to compute avg. bandwidth.

    Args:
      file_path: String path to GCS file starting from gs://.
      file_size: Integer denoting the size of file in bytes.
      block_size: Integer denoting the size of block in bytes to keep while
        reading.

    Returns:
      Integer denoting the size of content read in bytes.
    """
    content_len = 0
    with tf.io.gfile.GFile(file_path, 'rb') as fp:
      for _ in range(0, file_size, block_size):
        content = fp.read(block_size)
        content_len = content_len + len(content)
      fp.close()
    return content_len

  def create_files(self, num_processes):
    """Creates num_processes number of GCS files to be used in load testing.

    Args:
      num_processes: Integer denoting number of processes in load test.

    Returns:
      None
    """
    logging.info(
        'One file is created per process of size %s using the format '
        '%s', self.FILE_SIZE, self.FILE_PATH_FORMAT)
    for process_num in range(num_processes):
      file_path = self.FILE_PATH_FORMAT.format(process_num=process_num)
      _create_binary_file(file_path, self.FILE_SIZE)


# Disabling doc string pylint beyond this because it is not expected to write
# doc string for each task class when it is available for base class.
#pylint: disable=missing-class-docstring
class TFGFileRead256KB(TFGFileRead):

  TASK_NAME = '256kb'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/256kb/' \
                     'read.{process_num}.0'
  FILE_SIZE = 256 * 1024
  BLOCK_SIZE = 256 * 1024

  def task(self, assigned_process_id, assigned_thread_id):
    file_path = self.FILE_PATH_FORMAT.format(process_num=assigned_process_id)
    return self._tf_read_task(file_path, self.FILE_SIZE, self.BLOCK_SIZE)


class TFGFileRead3MB(TFGFileRead):

  TASK_NAME = '3mb'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/3mb/' \
                     'read.{process_num}.0'
  FILE_SIZE = 3 * 1024 * 1024
  BLOCK_SIZE = 3 * 1024 * 1024

  def task(self, assigned_process_id, assigned_thread_id):
    file_path = self.FILE_PATH_FORMAT.format(process_num=assigned_process_id)
    return self._tf_read_task(file_path, self.FILE_SIZE, self.BLOCK_SIZE)


class TFGFileRead5MB(TFGFileRead):

  TASK_NAME = '5mb'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/5mb/' \
                     'read.{process_num}.0'
  FILE_SIZE = 5 * 1024 * 1024
  BLOCK_SIZE = 5 * 1024 * 1024

  def task(self, assigned_process_id, assigned_thread_id):
    file_path = self.FILE_PATH_FORMAT.format(process_num=assigned_process_id)
    return self._tf_read_task(file_path, self.FILE_SIZE, self.BLOCK_SIZE)


class TFGFileRead50MB(TFGFileRead):

  TASK_NAME = '50mb'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/50mb/' \
                     'read.{process_num}.0'
  FILE_SIZE = 50 * 1024 * 1024
  BLOCK_SIZE = 50 * 1024 * 1024

  def task(self, assigned_process_id, assigned_thread_id):
    file_path = self.FILE_PATH_FORMAT.format(process_num=assigned_process_id)
    return self._tf_read_task(file_path, self.FILE_SIZE, self.BLOCK_SIZE)


class TFGFileRead100MB(TFGFileRead):

  TASK_NAME = '100mb'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/100mb/' \
                     'read.{process_num}.0'
  FILE_SIZE = 100 * 1024 * 1024
  BLOCK_SIZE = 100 * 1024 * 1024

  def task(self, assigned_process_id, assigned_thread_id):
    file_path = self.FILE_PATH_FORMAT.format(process_num=assigned_process_id)
    return self._tf_read_task(file_path, self.FILE_SIZE, self.BLOCK_SIZE)


class TFGFileRead200MB(TFGFileRead):

  TASK_NAME = '200mb'

  FILE_PATH_FORMAT = 'gs://load-test-bucket/python/files/200mb/' \
                     'read.{process_num}.0'
  FILE_SIZE = 200 * 1024 * 1024
  BLOCK_SIZE = 200 * 1024 * 1024

  def task(self, assigned_process_id, assigned_thread_id):
    file_path = self.FILE_PATH_FORMAT.format(process_num=assigned_process_id)
    return self._tf_read_task(file_path, self.FILE_SIZE, self.BLOCK_SIZE)
