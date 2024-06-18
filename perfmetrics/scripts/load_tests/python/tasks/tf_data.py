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
"""Contains tasks which reads data from files using tensorflow's tf.data api.

Example:

  lg_obj = load_generator.LoadGenerator(...)
  task_obj = tf_data.TFDataRead(...)
  observations = lg_obj.generate_load(task_obj)
"""
import os
import logging
import tensorflow as tf
from load_generator import task

# .tfrecord file contains multiple TFRecords in it. This defines the size of
# that single TFRecord in bytes.
SINGLE_RECORD_SIZE = 256 * 1024
TF_DATA_READ = 'tf_data_read'


class TFDataRead(task.LoadTestTask):
  """Task class for reading tfrecord file using tensorflow's tf.data api.

  tf.data: https://www.tensorflow.org/guide/data
  tf.data.TFRecordDataset.shard:
  https://www.tensorflow.org/api_docs/python/tf/data/Dataset#shard
  Note: The same class can be used with tf's internal GCS client if format of
  file path starts with gs:// and with GCSFuse if local GCSFuse mounted path
  is passed.

  For details on prefetch, num_parallel_calls & shard arguments, please refer
  to tf.data public documentation.

  Args:
    task_name: String name assigned to task.
    file_path_format: String format of file path. Must contain 'file_num' to be
      formatted. E.g. gcs/256kb/read.{file_num}.
    file_size: Integer size of file in bytes to be read.
    num_files: Integer number of files to be read in one task.
    prefetch: Integer value of prefetch. Default to AUTOTUNE(-1).
    num_parallel_calls = Integer value of num_parallel_calls. Default to
      AUTOTUNE(-1).
    shard: Integer value of shard. Default to AUTOTUNE(-1).
  """

  def __init__(self,
               task_name,
               file_path_format,
               file_size,
               num_files,
               prefetch=-1,
               num_parallel_calls=-1,
               shard=1):
    super().__init__()
    self.task_type = TF_DATA_READ
    self.task_name = task_name
    self.file_path_format = file_path_format
    self.file_size = file_size
    self.num_files = num_files
    self.prefetch = prefetch
    self.num_parallel_calls = num_parallel_calls
    self.shard = shard
    self.file_names = [
        self.file_path_format.format(file_num=file_num)
        for file_num in range(self.num_files)
    ]

  def pre_task(self, process_id, thread_id):  #pylint: disable=unused-argument
    """Pre-task to clear OS's page caches in load testing.
    """
    # Clear the page cache as there is no way to bypass cache in tf.data.
    # Note: This takes time and hence decrease the average bandwidth.
    os.system("sudo sh -c 'sync; echo 3 >  /proc/sys/vm/drop_caches'")

  def task(self, process_id, thread_id):  #pylint: disable=unused-argument
    """Reads TFRecord files using tensorflow's tf.data method.

    See base class for more details.

    Returns:
      Integer denoting the size of content read in bytes.
    """
    content_len = 0
    files_dataset = tf.data.Dataset.from_tensor_slices(self.file_names)

    def tfrecord(path):
      return tf.data.TFRecordDataset(path).prefetch(self.prefetch).shard(
          self.shard, 0)

    dataset = files_dataset.interleave(
        tfrecord,
        cycle_length=self.num_parallel_calls,
        num_parallel_calls=self.num_parallel_calls)
    for record in dataset:
      content_len = content_len + len(record.numpy())

    return content_len * self.shard

  def create_files(self, num_processes):  #pylint: disable=unused-argument
    """Creates self.num_files number of files to be used in load testing.
    """
    logging.info('Creating %s bytes TFRecord files using the format '
                 '%s', self.file_size, self.file_path_format)

    default_file_path = self.file_path_format.format(file_num=0)
    self._create_tfrecord_file(default_file_path, self.file_size)
    for file_num in range(1, self.num_files):
      file_path = self.file_path_format.format(file_num=file_num)
      if not tf.io.gfile.exists(file_path):
        logging.info('Creating TFRecord file %s of size %s.', file_path,
                     self.file_size)
        tf.io.gfile.copy(default_file_path, file_path)

  def _create_tfrecord_file(self, file_path, file_size):
    """Creates .tfrecord file of given file size in bytes at given file path.

    Doesn't create a file if file is already present.
    """
    # We only check existence in this case because actual TFRecord's file size
    # is not exactly equal to file_size
    if tf.io.gfile.exists(file_path):
      return

    logging.info('Creating TFRecord file %s of size %s.', file_path, file_size)
    content = b'\t' * SINGLE_RECORD_SIZE
    writer = tf.io.TFRecordWriter(file_path)
    for _ in range(0, file_size, len(content)):
      writer.write(content)
    writer.close()
