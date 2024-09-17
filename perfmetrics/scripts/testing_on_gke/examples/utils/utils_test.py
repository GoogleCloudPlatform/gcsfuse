"""This file defines unit tests for functionalities in utils.py"""

# Copyright 2018 The Kubernetes Authors.
# Copyright 2024 Google LLC
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

import unittest
import utils
from utils import get_cpu_from_monitoring_api, get_memory_from_monitoring_api, timestamp_to_epoch


class UtilsTest(unittest.TestCase):

  @classmethod
  def setUpClass(self):
    self.project_id = 'gcs-fuse-test'
    self.cluster_name = 'gargnitin-dryrun-us-west1-6'
    self.pod_name = 'fio-tester-gcsfuse-rr-64k-1670041227260535313'
    # self.container_name = "fio-tester"
    self.namespace_name = 'default'
    self.start_epoch = 1724233283
    self.end_epoch = 1724233442

  def test_get_memory_from_monitoring_api(self):
    print(
        get_memory_from_monitoring_api(
            self.project_id,
            self.cluster_name,
            self.pod_name,
            # self.container_name,
            self.namespace_name,
            self.start_epoch,
            self.end_epoch,
        )
    )

  def test_get_cpu_from_monitoring_api(self):
    print(
        get_cpu_from_monitoring_api(
            self.project_id,
            self.cluster_name,
            self.pod_name,
            # self.container_name,
            self.namespace_name,
            self.start_epoch,
            self.end_epoch,
        )
    )

  def test_timestamp_to_epoch(self):
    timestamp = '2024-08-21T19:20:25'
    expected_epoch = 1724268025
    self.assertEqual(timestamp_to_epoch(timestamp), expected_epoch)
    pass


if __name__ == '__main__':
  unittest.main()
