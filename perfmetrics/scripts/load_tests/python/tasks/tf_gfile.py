# Copyright 2023 Google Inc. All Rights Reserved.
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
"""Contains tasks that read data from files using tensorflow's tf.io.gfile api.

Example:

  lg_obj = load_generator.LoadGenerator(...)
  task_obj = python_os.TFGFileRead(...)
  observations = lg_obj.generate_load(task_obj)
"""
import logging
import tensorflow as tf
from load_generator import task

TF_GFILE_READ = 'tf_gfile_read'


class TFGFileRead(task.LoadTestTask):
  """Task class for reading GCS file using tensorflow's tf.io.gfile api.

  tf.io.gfile: https://www.tensorflow.org/api_docs/python/tf/io/gfile/GFile
  Note: The same class can be used with tf's internal GCS client if format of
  file path starts with gs:// and with GCSFuse if local GCSFuse mounted path
  is passed.

  Args:
    task_name: String name assigned to task.
    file_path_format: String format of file path. The format can contain
     'process_id' and 'thread_id' to be filled. E.g. gcs/256kb/read.{process_id}
    file_size: Integer size of file in bytes to be read.
    block_size: Integer size of block in bytes. File is read block by block.
      If block size is not passed or -1, then by default it takes file size as
      block size
  """

  def __init__(self, task_name, file_path_format, file_size, block_size=-1):
    super().__init__()
    self.task_type = TF_GFILE_READ
    self.task_name = task_name
    self.file_path_format = file_path_format
    self.file_size = file_size
    # Keep file size as default block size
    self.block_size = block_size
    if self.block_size == -1:
      self.block_size = file_size

  def task(self, process_id, thread_id):
    """Reads file of given size from given GCS file path and block size.

    See base class for more details.

    Returns:
      Integer denoting the size of content read in bytes.
    """
    file_path = self.file_path_format.format(
        process_id=process_id, thread_id=thread_id)
    content_len = 0
    with tf.io.gfile.GFile(file_path, 'rb') as fp:
      for _ in range(0, self.file_size, self.block_size):
        content = fp.read(self.block_size)
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
        '%s', self.file_size, self.file_path_format)
    for process_id in range(num_processes):
      file_path = self.file_path_format.format(process_id=process_id)
      self._create_binary_file(file_path, self.file_size)

  def _create_binary_file(self, file_path, file_size):
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
