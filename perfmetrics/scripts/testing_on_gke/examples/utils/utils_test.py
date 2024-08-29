"""This file defines unit tests for functionalities in utils.py"""

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

  def test_resource_limit(self):
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
      self.assertEqual(dict, type(input))
      try:
        resource_limits = utils.resource_limits(input['nodeType'])
        self.assertEqual(
            input['expected_limits_cpu'],
            resource_limits[0]['cpu'],
        )
        self.assertFalse(input['expected_error'])
      except utils.UnknownMachineTypeError:
        self.assertTrue(input['expected_error'])
    pass


if __name__ == '__main__':
  unittest.main()
