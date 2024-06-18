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
"""Contains task which reads data from files using python's native open api.

Example:

  lg_obj = load_generator.LoadGenerator(...)
  task_obj = python_os.OSRead256KB(...)
  observations = lg_obj.generate_load(task_obj)
"""
import os
import logging
from load_generator import task

PYTHON_OS_READ = 'python_os_read'


class OSRead(task.LoadTestTask):
  """Task class for reading file from disk using python's native open api.

  The task reads file with os.O_DIRECT i.e. bypassing page cache.

  Args:
    task_name: String name assigned to task.
    file_path_format: String format of file path. The format can contain
     'process_id' and 'thread_id' to be filled. E.g. gcs/256kb/read.{process_id}
    file_size: Integer size of file in bytes to be read.
    block_size: Integer size of block in bytes. File is read block by block.
      If block size is not passed or -1, then by default it takes file size as
      block size.
  """

  def __init__(self, task_name, file_path_format, file_size, block_size=-1):
    super().__init__()
    self.task_type = PYTHON_OS_READ
    self.task_name = task_name
    self.file_path_format = file_path_format
    self.file_size = file_size
    # Keep file size as default block size
    self.block_size = block_size
    if self.block_size == -1:
      self.block_size = file_size

  def task(self, process_id, thread_id):
    """Reads file of given size from given file path and with given block size.

    See base class for more details.

    Returns:
      Integer denoting the size of content read in bytes.
    """
    file_path = self.file_path_format.format(
        process_id=process_id, thread_id=thread_id)
    my_file = os.open(file_path, os.O_DIRECT)
    content_len = 0
    with open(my_file, 'rb') as f_p:
      for _ in range(0, self.file_size, self.block_size):
        content = f_p.read(self.block_size)
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
    if not os.path.exists(os.path.dirname(self.file_path_format)):
      raise RuntimeError('Directory containing files for task not exists.')

    logging.info(
        'One file is created per process of size %s using the format '
        '%s', self.file_size, self.file_path_format)
    for process_id in range(num_processes):
      file_path = self.file_path_format.format(process_id=process_id)
      self._create_binary_file(file_path, self.file_size)

  def _create_binary_file(self, file_path, file_size):
    """Creates binary file of given file size in bytes at given path.

    Doesn't create a file if file is already present and has same file size.
    """
    if os.path.exists(file_path) and os.path.getsize(file_path) == file_size:
      return
    logging.info('Creating file %s of size %s.', file_path, file_size)
    with open(file_path, 'wb') as f_p:
      f_p.truncate(file_size)
