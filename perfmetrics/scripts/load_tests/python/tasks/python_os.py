"""Contains tasks which reads data from files using python's native open api.

Contains tasks for reading file sizes: 256KiB, 3MiB, 5MiB, 50MiB, 100MiB,
200MiB. OSRead is the base class for all the tasks.

Example:

  lg_obj = load_generator.LoadGenerator(...)
  task_obj = python_os.OSRead256KB()
  observations = lg_obj.generate_load(task_obj)
"""
import os
import logging
from load_generator import task


def _create_binary_file(file_path, file_size):
  """Creates binary file of given file size in bytes at given path.

  Doesn't create a file if file is already present and has same file size.
  """
  if os.path.exists(file_path) and os.path.getsize(file_path) == file_size:
    return
  logging.info('Creating file %s of size %s.', file_path, file_size)
  with open(file_path, 'wb') as f_p:
    f_p.truncate(file_size)


class OSRead(task.LoadTestTask):
  """Base task class for reading file from disk using python's native open api.

  Note: (a) This class is abstract because it doesn't define task method.
  (b) It is mandatory to define task method, keep TASK_TYPE = 'read' and to
  assign appropriate values of FILE_PATH_FORMAT, FILE_SIZE and BLOCK_SIZE in
  derived classes.
  """
  TASK_TYPE = 'read'

  FILE_PATH_FORMAT = ''
  FILE_SIZE = 0
  BLOCK_SIZE = 0

  def _os_direct_read_task(self, file_path, file_size, block_size):
    """Reads file of given size from given file path and with given block size.

    Note: It is important to return length of content read because the returned
    value is used by post task of load generator to compute avg. bandwidth.

    Args:
      file_path: String path to file.
      file_size: Integer denoting the size of file in bytes.
      block_size: Integer denoting the size of block in bytes to keep while
        reading.

    Returns:
      Integer denoting the size of content read in bytes.
    """
    my_file = os.open(file_path, os.O_DIRECT)
    content_len = 0
    with open(my_file, 'rb') as f_p:
      for _ in range(0, file_size, block_size):
        content = f_p.read(block_size)
        content_len = content_len + len(content)
      f_p.close()
    return content_len

  def create_files(self, num_processes):
    """Creates num_processes number of files to be used in load testing.

    Args:
      num_processes: Integer denoting number of processes in load test.

    Returns:
      None

    Raises:
      RuntimeError: If the path under which the files to be created doesn't
        exist.
    """
    # Create one file per process for read and write tasks.
    if not os.path.exists(os.path.dirname(self.FILE_PATH_FORMAT)):
      raise RuntimeError('Directory containing files for task not exists.')

    logging.info(
        'One file is created per process of size %s using the format '
        '%s', self.FILE_SIZE, self.FILE_PATH_FORMAT)
    for process_num in range(num_processes):
      file_path = self.FILE_PATH_FORMAT.format(process_num=process_num)
      _create_binary_file(file_path, self.FILE_SIZE)


# Disabling doc string pylint beyond this because it is not expected to write
# doc string for each task class when it is available for base class.
#pylint: disable=missing-class-docstring
class OSRead256KB(OSRead):

  TASK_NAME = '256kb'

  FILE_PATH_FORMAT = '/gcs/256kb/read.{process_num}.0'
  FILE_SIZE = 256 * 1024
  BLOCK_SIZE = 256 * 1024

  def task(self, assigned_process_id, assigned_thread_id):
    file_path = self.FILE_PATH_FORMAT.format(process_num=assigned_process_id)
    return self._os_direct_read_task(file_path, self.FILE_SIZE, self.BLOCK_SIZE)


class OSRead3MB(OSRead):

  TASK_NAME = '3mb'

  FILE_PATH_FORMAT = '/gcs/3mb/read.{process_num}.0'
  FILE_SIZE = 3 * 1024 * 1024
  BLOCK_SIZE = 3 * 1024 * 1024

  def task(self, assigned_process_id, assigned_thread_id):
    file_path = self.FILE_PATH_FORMAT.format(process_num=assigned_process_id)
    return self._os_direct_read_task(file_path, self.FILE_SIZE, self.BLOCK_SIZE)


class OSRead5MB(OSRead):

  TASK_NAME = '5mb'

  FILE_PATH_FORMAT = '/gcs/5mb/read.{process_num}.0'
  FILE_SIZE = 5 * 1024 * 1024
  BLOCK_SIZE = 5 * 1024 * 1024

  def task(self, assigned_process_id, assigned_thread_id):
    file_path = self.FILE_PATH_FORMAT.format(process_num=assigned_process_id)
    return self._os_direct_read_task(file_path, self.FILE_SIZE, self.BLOCK_SIZE)


class OSRead50MB(OSRead):

  TASK_NAME = '50mb'

  FILE_PATH_FORMAT = '/gcs/50mb/read.{process_num}.0'
  FILE_SIZE = 50 * 1024 * 1024
  BLOCK_SIZE = 50 * 1024 * 1024

  def task(self, assigned_process_id, assigned_thread_id):
    file_path = self.FILE_PATH_FORMAT.format(process_num=assigned_process_id)
    return self._os_direct_read_task(file_path, self.FILE_SIZE, self.BLOCK_SIZE)


class OSRead100MB(OSRead):

  TASK_NAME = '100mb'

  FILE_PATH_FORMAT = '/gcs/100mb/read.{process_num}.0'
  FILE_SIZE = 100 * 1024 * 1024
  BLOCK_SIZE = 100 * 1024 * 1024

  def task(self, assigned_process_id, assigned_thread_id):
    file_path = self.FILE_PATH_FORMAT.format(process_num=assigned_process_id)
    return self._os_direct_read_task(file_path, self.FILE_SIZE, self.BLOCK_SIZE)


class OSRead200MB(OSRead):

  TASK_NAME = '200mb'

  FILE_PATH_FORMAT = '/gcs/200mb/read.{process_num}.0'
  FILE_SIZE = 200 * 1024 * 1024
  BLOCK_SIZE = 200 * 1024 * 1024

  def task(self, assigned_process_id, assigned_thread_id):
    file_path = self.FILE_PATH_FORMAT.format(process_num=assigned_process_id)
    return self._os_direct_read_task(file_path, self.FILE_SIZE, self.BLOCK_SIZE)
