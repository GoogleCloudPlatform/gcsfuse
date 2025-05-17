# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""This file defines tests for functionalities in bq_utils.py"""

import calendar
import copy
from random import randrange, uniform
import sys
import time
import unittest

sys.path.append('../')
from utils.utils import unix_to_timestamp
from bq_utils import FioBigqueryExporter, FioTableRow, Timestamp


class BqUtilsTest(unittest.TestCase):

  @classmethod
  def setUpClass(self):
    self.bq_project_id = 'gcs-fuse-test-ml'
    self.bq_dataset_id = 'gargnitin_test_gke_test_tool_outputs'
    self.bq_table_id = 'fio_outputs_test'
    # Create a sample table for manual testing.
    self.fioBqExporter = FioBigqueryExporter(
        self.bq_project_id, self.bq_dataset_id, self.bq_table_id
    )

  @classmethod
  def cur_epoch(self) -> int:
    return int(calendar.timegm(time.gmtime()))

  @classmethod
  def cur_timestamp(self) -> Timestamp:
    return unix_to_timestamp(self.cur_epoch())

  @classmethod
  def create_sample_fio_table_row(self):
    row = FioTableRow()
    row.fio_workload_id = 'fio-workload-{randrange(10000000000)}'
    row.experiment_id = f'expt-{self.cur_epoch()}'
    row.epoch = 1
    row.file_size = '1M'
    row.file_size_in_bytes = 2**20
    row.block_size = '256K'
    row.block_size_in_bytes = 2**18
    row.bucket_name = 'sample-zb-bucket'
    row.e2e_latency_ns_p50 = randrange(1, 1000)
    row.e2e_latency_ns_p90 = randrange(row.e2e_latency_ns_p50, 10000)
    row.e2e_latency_ns_p99 = randrange(row.e2e_latency_ns_p90, 100000)
    row.e2e_latency_ns_p99_9 = randrange(row.e2e_latency_ns_p99, 1000000)
    row.e2e_latency_ns_max = randrange(row.e2e_latency_ns_p99_9, 10000000)
    row.start_epoch = self.cur_epoch()
    row.duration_in_seconds = randrange(1, 60)
    row.end_epoch = row.start_epoch + row.duration_in_seconds
    row.files_per_thread = 20000
    row.gcsfuse_mount_options = 'implicit-dirs'
    row.lowest_cpu_usage = uniform(1.0, 100.0)
    row.highest_cpu_usage = uniform(row.lowest_cpu_usage, 100.0)
    row.lowest_memory_usage = uniform(10.0, 10000.0)
    row.highest_memory_usage = uniform(row.lowest_memory_usage, 10000.0)
    row.iops = uniform(10.0, 10000.0)
    row.machine_type = 'n2-standard-32'
    row.num_threads = 50
    row.operation = 'read'
    row.pod_name = f'sample-pod-name-{randrange(100000000000)}'
    row.scenario = 'gcsfuse-generic'
    row.start_time = self.cur_timestamp()
    row.end_time = row.start_time
    row.throughput_in_mbps = uniform(1.0, 10000.0)
    return row

  def test_insert_multiple_rows(self):
    rows = []
    orig_num_rows = self.fioBqExporter._num_rows()

    rowCommon = self.create_sample_fio_table_row()

    row = copy.deepcopy(rowCommon)
    row.fio_workload_id = f'fio_workload1_{row.experiment_id}'
    row.epoch = 1
    row.start_time = Timestamp('2025-05-09 08:31 UTC')
    row.end_time = row.start_time
    rows.append(row)

    row = copy.deepcopy(row)
    row.fio_workload_id = f'fio_workload2_{row.experiment_id}'
    row.epoch = 1
    row.start_time = Timestamp('2025-05-09 08:32 UTC')
    row.end_time = row.start_time
    rows.append(row)

    self.fioBqExporter.insert_rows(rows)
    self.assertEqual(self.fioBqExporter._num_rows(), orig_num_rows + 2)

  def test_insert_rows_with_one_bad_row(self):
    rows = []
    orig_num_rows = self.fioBqExporter._num_rows()

    rowCommon = self.create_sample_fio_table_row()

    row = copy.deepcopy(rowCommon)
    num_rows = 20
    for i in range(num_rows):
      row = copy.deepcopy(row)
      row.fio_workload_id = f'fio_workload{i}_{row.experiment_id}'
      row.epoch = 1
      if i == 0:
        # First row is bad row because of empty start_time and end_time.
        row.start_time = Timestamp('')
        row.end_time = Timestamp('')
      else:
        row.start_time = self.cur_timestamp()
        row.end_time = row.start_time
      rows.append(row)

    # Despite bad row(s), the insert_rows itself will not fail
    # because of the fallback in insert_rows.
    self.fioBqExporter.insert_rows(rows)
    self.assertEqual(
        self.fioBqExporter._num_rows(), orig_num_rows + num_rows - 1
    )

  def test_num_rows(self):
    row = self.create_sample_fio_table_row()
    orig_num_rows = self.fioBqExporter._num_rows()

    self.fioBqExporter.insert_rows([row])

    self.assertEqual(self.fioBqExporter._num_rows(), orig_num_rows + 1)

  def test_has_experiment_id(self):
    row = self.create_sample_fio_table_row()

    self.fioBqExporter.insert_rows([row])

    # _has_experiment_id should return true for an experiment_id
    # which has already been added to the table.
    self.assertTrue(self.fioBqExporter._has_experiment_id(row.experiment_id))

  def test_has_experiment_id_negative(self):
    # _has_experiment_id should return false for an experiment_id
    # which has not been added to the table.
    self.assertFalse(
        self.fioBqExporter._has_experiment_id('invalid-experiment-id')
    )

  def test_insert_rows_with_unset_experiment_id(self):
    row = self.create_sample_fio_table_row()
    row.experiment_id = None

    with self.assertRaises(Exception):
      self.fioBqExporter.insert_rows([row])

  def test_insert_rows_with_mismatched_experiment_ids(self):
    row1 = self.create_sample_fio_table_row()
    row1.experiment_id = 'expt-id-1'
    row2 = copy.deepcopy(row1)
    row2.experiment_id = 'expt-id-2'

    with self.assertRaises(Exception):
      self.fioBqExporter.insert_rows([row1, row2])


if __name__ == '__main__':
  unittest.main()
