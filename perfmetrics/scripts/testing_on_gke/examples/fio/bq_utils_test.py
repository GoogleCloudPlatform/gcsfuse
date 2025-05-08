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

import unittest
from bq_utils import FioBigqueryExporter, FioTableRow, Timestamp
import utils


class BqUtilsTest(unittest.TestCase):

  @classmethod
  def setUpClass(self):
    self.bq_project_id = 'gcs-fuse-test-ml'
    self.bq_dataset_id = 'gargnitin_test_gke_test_tool_outputs'
    self.bq_table_id = 'fio_outputs_test'

  def test_create_bq_table(self):
    # Create a sample table for manual testing.
    fioBqExporter = FioBigqueryExporter(
        self.bq_project_id, self.bq_dataset_id, self.bq_table_id
    )

    # sample append call.
    rows = []

    row = FioTableRow()
    row.fio_workload_id = 'fio-workload-id-1'
    row.experiment_id = 'expt-id-1'
    row.epoch = 1
    row.file_size = '1M'
    row.file_size_in_bytes = 2**20
    row.block_size = '256K'
    row.block_size_in_bytes = 2**18
    row.bucket_name = 'sample-zb-bucket'
    row.duration_in_seconds = 10
    row.e2e_latency_ns_max = 100
    row.e2e_latency_ns_p50 = 50
    row.e2e_latency_ns_p90 = 90
    row.e2e_latency_ns_p99 = 99
    row.e2e_latency_ns_p99_9 = 99.9
    row.end_epoch = 1746678693
    row.start_epoch = 1746678683
    row.files_per_thread = 20000
    row.gcsfuse_mount_options = 'implicit-dirs'
    row.highest_cpu_usage = 10.0
    row.lowest_cpu_usage = 1.0
    row.highest_memory_usage = 10000
    row.lowest_memory_usage = 100
    row.iops = 1000
    row.machine_type = 'n2-standard-32'
    row.num_threads = 50
    row.operation = 'read'
    row.pod_name = 'sample-pod-name'
    row.scenario = 'gcsfuse-generic'
    row.start_time = Timestamp('2025-05-08 04:31:23 UTC')
    row.end_time = Timestamp('2025-05-08 04:31:33 UTC')
    row.throughput_in_mbps = 8000

    rows.append(row)
    fioBqExporter.append_rows(rows)


if __name__ == '__main__':
  unittest.main()
