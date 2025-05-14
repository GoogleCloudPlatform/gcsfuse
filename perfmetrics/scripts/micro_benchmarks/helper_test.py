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


class TestSetupFunctions(unittest.TestCase):
  @patch("helper.subprocess.run")
  @patch("helper.os.makedirs")
  def test_mount_bucket(self, mock_makedirs, mock_subprocess_run):
    helper.mount_bucket("/mnt/gcs", "my-bucket", "--implicit-dirs")
    mock_makedirs.assert_called_once_with("/mnt/gcs", exist_ok=True)
    mock_subprocess_run.assert_called_once_with(
        "gcsfuse --implicit-dirs my-bucket /mnt/gcs",
        shell=True,
        check=True
    )

  @patch("helper.subprocess.run")
  def test_unmount_gcs_directory_success(self, mock_subprocess_run):
    helper.unmount_gcs_directory("/mnt/gcs")
    mock_subprocess_run.assert_called_once_with(["fusermount", "-u", "/mnt/gcs"], check=True)

  @patch("helper.bigquery.Client")  # Only patch bigquery.Client
  def test_log_to_bigquery(self, mock_bq_client):
      duration = 10.0
      total_bytes = 100 * 1000 * 1000  # 100 MB
      flags = "--implicit-dirs"
      workload_type = "write"

      # Setup BigQuery mock
      mock_bq_instance = MagicMock()
      mock_bq_client.return_value = mock_bq_instance
      mock_dataset = mock_bq_instance.dataset.return_value
      mock_table = mock_dataset.table.return_value
      mock_bq_instance.load_table_from_dataframe.return_value.result.return_value = None

      # Act
      helper.log_to_bigquery(duration, total_bytes, flags, workload_type)

      # Assert
      mock_bq_instance.dataset.assert_called_with("benchmark_results")
      mock_dataset.table.assert_called_with("gcsfuse_benchmarks")
      mock_bq_instance.load_table_from_dataframe.assert_called_once()
      mock_bq_instance.load_table_from_dataframe().result.assert_called_once()

if __name__ == "__main__":
  unittest.main()
