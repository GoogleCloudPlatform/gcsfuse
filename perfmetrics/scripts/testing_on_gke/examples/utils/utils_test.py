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
from utils import get_cpu, get_cpu_from_monitoring_api, get_memory, get_memory_from_monitoring_api, timestamp_to_epoch


class UtilsTest(unittest.TestCase):

  @classmethod
  def setUpClass(self):
    self.project_id = 'gcs-fuse-test-ml'
    self.project_number = 786757290066
    self.cluster_name = 'gargnitin-gketesting-us-west1b'
    self.pod_name = 'fio-tester-gcsfuse-rr-200g-5061221202711042867'
    self.namespace_name = 'default'
    self.start_epoch = 1727245942
    self.end_epoch = 1727247982
    self.start = '2024-09-25 06:32:22 UTC'
    self.end = '2024-09-25 07:06:22 UTC'

  def test_get_memory_methods(self):
    low1, high1 = get_memory_from_monitoring_api(
        project_id=self.project_id,
        cluster_name=self.cluster_name,
        pod_name=self.pod_name,
        namespace_name=self.namespace_name,
        start_epoch=self.start_epoch,
        end_epoch=self.end_epoch,
    )
    self.assertLessEqual(low1, high1)
    self.assertGreater(high1, 0)

    low2, high2 = get_memory(
        project_number=self.project_number,
        pod_name=self.pod_name,
        start=self.start,
        end=self.end,
    )
    self.assertLessEqual(low2, high2)
    self.assertGreater(high2, 0)

    self.assertTrue(high1 >= 0.99 * high2 and high1 <= 1.01 * high2)

  def test_get_cpu_methods(self):
    low1, high1 = get_cpu_from_monitoring_api(
        project_id=self.project_id,
        cluster_name=self.cluster_name,
        pod_name=self.pod_name,
        namespace_name=self.namespace_name,
        start_epoch=self.start_epoch,
        end_epoch=self.end_epoch,
    )
    self.assertLessEqual(low1, high1)
    self.assertGreater(high1, 0)

    low2, high2 = get_cpu(
        project_number=self.project_number,
        pod_name=self.pod_name,
        start=self.start,
        end=self.end,
    )
    self.assertLessEqual(low2, high2)
    self.assertGreater(high2, 0)

    self.assertTrue(high1 >= 0.99 * high2 and high1 <= 1.01 * high2)

  def test_timestamp_to_epoch(self):
    self.assertEqual(timestamp_to_epoch('2024-08-21T19:20:25'), 1724268025)

  def test_timestamp_to_epoch_with_nznano(self):
    self.assertEqual(
        timestamp_to_epoch('2024-08-21T19:20:25.547456'), 1724268025
    )


if __name__ == '__main__':
  unittest.main()
