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
from utils import convert_size_to_bytes, get_cpu_from_monitoring_api, get_memory_from_monitoring_api, timestamp_to_epoch


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

  def test_get_memory_from_monitoring_api(self):
    low, high = get_memory_from_monitoring_api(
        project_id=self.project_id,
        cluster_name=self.cluster_name,
        pod_name=self.pod_name,
        namespace_name=self.namespace_name,
        start_epoch=self.start_epoch,
        end_epoch=self.end_epoch,
    )

    self.assertLessEqual(low, high)
    self.assertGreater(high, 0)

  def test_get_cpu_from_monitoring_api(self):
    low, high = get_cpu_from_monitoring_api(
        project_id=self.project_id,
        cluster_name=self.cluster_name,
        pod_name=self.pod_name,
        namespace_name=self.namespace_name,
        start_epoch=self.start_epoch,
        end_epoch=self.end_epoch,
    )

    self.assertLessEqual(low, high)
    self.assertGreater(high, 0)

  def test_timestamp_to_epoch(self):
    self.assertEqual(timestamp_to_epoch('2024-08-21T19:20:25'), 1724268025)

  def test_timestamp_to_epoch_with_nznano(self):
    self.assertEqual(
        timestamp_to_epoch('2024-08-21T19:20:25.547456'), 1724268025
    )

  def test_resource_limits(self):
    inputs = [
        {
            'nodeType': 'n2-standard-32',
            'expected_limits_cpu': 32,
            'expected_error': False,
        },
        {
            'nodeType': 'n2-standard-96',
            'expected_limits_cpu': 96,
            'expected_error': False,
        },
        {
            'nodeType': 'n2-standard-96',
            'expected_limits_cpu': 96,
            'expected_error': False,
        },
        {
            'nodeType': 'c3-standard-176',
            'expected_limits_cpu': 176,
            'expected_error': False,
        },
        {
            'nodeType': 'c3-standard-176-lssd',
            'expected_limits_cpu': 176,
            'expected_error': False,
        },
        {'nodeType': 'n2-standard-1', 'expected_error': True},
        {'nodeType': 'unknown-machine-type', 'expected_error': True},
    ]

    for input in inputs:

      if input['expected_error']:

        with self.assertRaises(utils.UnknownMachineTypeError):
          utils.resource_limits(input['nodeType'])
      else:
        resource_limits = utils.resource_limits(input['nodeType'])

        self.assertEqual(
            input['expected_limits_cpu'],
            resource_limits[0]['cpu'],
        )

  def test_convert_size_to_bytes(self):
    inputs = {
        '': 0,
        '1': 1,
        '-1': -1,
        '0': 0,
        'k': 1000,
        'm': 1000000,
        'g': 1000000000,
        'K': 1024,
        'M': 1048576,
        'G': 1073741824,
        '1k': 1000,
        '1m': 1000000,
        '1g': 1000000000,
        '1K': 1024,
        '1M': 1048576,
        '1G': 1073741824,
        '2k': 2000,
        '2m': 2000000,
        '2g': 2000000000,
        '2K': 2048,
        '2M': 2097152,
        '2G': 2147483648,
    }
    for input, output in inputs.items():
      self.assertEqual(
          convert_size_to_bytes(input),
          output,
          f'Failed to convert for input = "{input}", expected-output ='
          f' {output}',
      )


if __name__ == '__main__':
  unittest.main()
