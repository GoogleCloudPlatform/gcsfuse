# Copyright 2025 Google LLC
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

import unittest
from unittest.mock import patch, MagicMock
import helper
import subprocess

class TestHelperFunctions(unittest.TestCase):
  @patch("helper.subprocess.run")
  @patch("helper.os.makedirs")
  def test_mount_bucket_success(self, mock_makedirs, mock_run):
    result = helper.mount_bucket("/mnt/success", "bucket", "--flags")

    self.assertTrue(result)

  @patch("helper.subprocess.run", side_effect=subprocess.CalledProcessError(1, "gcsfuse"))
  @patch("helper.os.makedirs")
  def test_mount_bucket_failure(self, mock_makedirs, mock_run):
    result = helper.mount_bucket("/mnt/fail", "bucket", "--flags")

    self.assertFalse(result)

  @patch('subprocess.run')
  def test_unmount_success(self, mock_run):
    mount_point = "/mnt/gcs_test"
    mock_run.return_value = subprocess.CompletedProcess(args=["fusermount", "-u", mount_point], returncode=0)

    result = helper.unmount_gcs_directory(mount_point)

    mock_run.assert_called_once_with(["fusermount", "-u", mount_point], check=True)
    self.assertTrue(result)

  @patch('subprocess.run', side_effect=subprocess.CalledProcessError(1, ["fusermount", "-u", "/mnt/gcs_test"]))
  def test_unmount_failure(self, mock_run):
    mount_point = "/mnt/gcs_test"

    result = helper.unmount_gcs_directory(mount_point)

    mock_run.assert_called_once_with(["fusermount", "-u", mount_point], check=True)
    self.assertFalse(result)

  @patch("helper.bigquery.Client")  # Only patch bigquery.Client
  def test_log_to_bigquery(self, mock_bq_client):
    duration = 10
    total_bytes = 100 * 1000 * 1000  # 100 MB
    flags = "--implicit-dirs"
    workload_type = "write"
    # Setup BigQuery mock
    mock_bq_instance = MagicMock()
    mock_bq_client.return_value = mock_bq_instance
    mock_dataset = mock_bq_instance.dataset.return_value
    mock_bq_instance.load_table_from_dataframe.return_value.result.return_value = None

    helper.log_to_bigquery(duration, total_bytes, flags, workload_type)

    mock_bq_instance.dataset.assert_called_with("benchmark_results")
    mock_dataset.table.assert_called_with("gcsfuse_benchmarks")
    mock_bq_instance.load_table_from_dataframe.assert_called_once()
    mock_bq_instance.load_table_from_dataframe().result.assert_called_once()

  @patch("helper.bigquery.Client")
  def test_log_to_bigquery_failure(self, mock_client_cls):
    # Create a mock client and a mock load job
    mock_client = MagicMock()
    mock_load_job = MagicMock()
    mock_client.load_table_from_dataframe.return_value = mock_load_job
    # Simulate failure on .result()
    mock_load_job.result.side_effect = Exception("BigQuery load failed")
    # Assign our mock client
    mock_client_cls.return_value = mock_client

    with self.assertRaises(Exception) as context:
      helper.log_to_bigquery(
          duration_sec=10.0,
          total_bytes=100_000_000,
          gcsfuse_config="--implicit-dirs",
          workload_type="read"
    )

    self.assertIn("BigQuery load failed", str(context.exception))

if __name__ == "__main__":
  unittest.main()
