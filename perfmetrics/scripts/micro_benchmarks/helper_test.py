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

from datetime import datetime
import subprocess
import unittest
from unittest.mock import patch, MagicMock
import helper

class TestBenchmarkFunctions(unittest.TestCase):

  @patch('subprocess.run')
  @patch('os.makedirs')
  def test_mount_bucket_success(self, mock_makedirs, mock_subprocess_run):
    """
    Tests that mount_bucket successfully calls subprocess.run and os.makedirs.
    """
    mount_dir = "/mnt/gcs_test"
    bucket_name = "my-test-bucket"
    flags = "--implicit-dirs"

    helper.mount_bucket(mount_dir, bucket_name, flags)

    # Assert that os.makedirs was called with the correct arguments
    mock_makedirs.assert_called_once_with(mount_dir, exist_ok=True)

    # Assert that subprocess.run was called with the correct command
    expected_cmd = f"gcsfuse {flags} {bucket_name} {mount_dir}"
    mock_subprocess_run.assert_called_once_with(expected_cmd, shell=True, check=True)

  @patch('subprocess.run')
  def test_unmount_gcs_directory_success(self, mock_subprocess_run):
    """
    Tests that unmount_gcs_directory successfully calls fusermount -u.
    """
    mount_point = "/mnt/gcs_test"

    helper.unmount_gcs_directory(mount_point)

    # Assert that subprocess.run was called with the correct command
    mock_subprocess_run.assert_called_once_with(["fusermount", "-u", mount_point], check=True)

  @patch('subprocess.run', side_effect=subprocess.CalledProcessError(1, 'fusermount -u'))
  def test_unmount_gcs_directory_failure(self, mock_subprocess_run):
    """
    Tests that unmount_gcs_directory handles CalledProcessError gracefully.
    """
    mount_point = "/mnt/gcs_test"

    # Capture print output to verify the error message
    with self.assertLogs(level='INFO') as cm: # Using assertLogs to capture print output
      helper.unmount_gcs_directory(mount_point)
      self.assertIn("❌ Failed to unmount", cm.output[0]) # Check for the error message

    # Assert that subprocess.run was still called
    mock_subprocess_run.assert_called_once_with(["fusermount", "-u", mount_point], check=True)

  @patch('google.cloud.bigquery.Client')
  @patch('pandas.DataFrame')
  @patch('pandas.to_datetime')
  def test_log_to_bigquery_success(self, mock_to_datetime, mock_dataframe, mock_bigquery_client):
    """
    Tests that log_to_bigquery correctly calculates bandwidth and calls BigQuery client methods.
    """
    duration_sec = 10.0
    total_bytes = 1024 * 1024 * 100  # 100 MB
    gcsfuse_config = "--max-conns=5"
    workload_type = "read"

    # Mock the BigQuery client and its methods
    mock_client_instance = mock_bigquery_client.return_value
    mock_table_ref = MagicMock()
    mock_client_instance.dataset.return_value.table.return_value = mock_table_ref
    mock_load_table_from_dataframe_result = MagicMock()
    mock_client_instance.load_table_from_dataframe.return_value.result.return_value = mock_load_table_from_dataframe_result

    # Mock the pandas DataFrame and its methods
    mock_df_instance = MagicMock()
    mock_dataframe.return_value = mock_df_instance
    mock_df_instance.__setitem__ = MagicMock() # Mock setting items (like df['timestamp'] = ...)
    mock_df_instance.astype = MagicMock(return_value=mock_df_instance) # Mock astype returning self

    # Mock pandas.to_datetime
    mock_to_datetime.return_value = MagicMock()

    helper.log_to_bigquery(duration_sec, total_bytes, gcsfuse_config, workload_type)

    # Assert BigQuery client and table reference are initialized correctly
    mock_bigquery_client.assert_called_once_with(project=PROJECT_ID)
    mock_client_instance.dataset.assert_called_once_with(DATASET_ID)
    mock_client_instance.dataset.return_value.table.assert_called_once_with(TABLE_ID)

    # Assert pandas.DataFrame was called with expected data
    # We can't directly check the exact datetime object, so we'll check the structure
    mock_dataframe.assert_called_once()
    args, kwargs = mock_dataframe.call_args
    self.assertIsInstance(args[0][0]['timestamp'], datetime)
    self.assertAlmostEqual(args[0][0]['duration_seconds'], duration_sec)
    # Calculate expected bandwidth
    expected_bandwidth_mbps = total_bytes / duration_sec / 1024 / 1024
    self.assertAlmostEqual(args[0][0]['bandwidth_mbps'], expected_bandwidth_mbps)
    self.assertEqual(args[0][0]['gcsfuse_config'], gcsfuse_config)
    self.assertEqual(args[0][0]['workload_type'], workload_type)

    # Assert type conversions were called
    mock_df_instance.__setitem__.assert_any_call('timestamp', mock_to_datetime.return_value)
    mock_df_instance.__setitem__.assert_any_call('duration_seconds', mock_df_instance.astype.return_value)
    mock_df_instance.__setitem__.assert_any_call('bandwidth_mbps', mock_df_instance.astype.return_value)
    mock_df_instance.astype.assert_any_call(float) # Called twice for duration and bandwidth

    # Assert load_table_from_dataframe was called with the mocked DataFrame and table reference
    mock_client_instance.load_table_from_dataframe.assert_called_once_with(mock_df_instance, mock_table_ref)
    mock_client_instance.load_table_from_dataframe.return_value.result.assert_called_once()

  @patch('google.cloud.bigquery.Client')
  @patch('pandas.DataFrame')
  @patch('pandas.to_datetime')
  def test_log_to_bigquery_bandwidth_calculation(self, mock_to_datetime, mock_dataframe, mock_bigquery_client):
    """
    Specifically tests the bandwidth calculation within log_to_bigquery.
    """
    duration_sec = 5.0
    total_bytes = 50 * 1024 * 1024  # 50 MB
    gcsfuse_config = ""
    workload_type = "write"

    # Mock the BigQuery client and its methods
    mock_client_instance = mock_bigquery_client.return_value
    mock_table_ref = MagicMock()
    mock_client_instance.dataset.return_value.table.return_value = mock_table_ref
    mock_load_table_from_dataframe_result = MagicMock()
    mock_client_instance.load_table_from_dataframe.return_value.result.return_value = mock_load_table_from_dataframe_result

    # Mock the pandas DataFrame and its methods
    mock_df_instance = MagicMock()
    mock_dataframe.return_value = mock_df_instance
    mock_df_instance.__setitem__ = MagicMock()
    mock_df_instance.astype = MagicMock(return_value=mock_df_instance)

    # Mock pandas.to_datetime
    mock_to_datetime.return_value = MagicMock()

    helper.log_to_bigquery(duration_sec, total_bytes, gcsfuse_config, workload_type)

    # Expected bandwidth calculation
    expected_bandwidth_mbps = total_bytes / duration_sec / 1024 / 1024

    # Assert pandas.DataFrame was called with expected data, specifically checking bandwidth
    mock_dataframe.assert_called_once()
    args, kwargs = mock_dataframe.call_args
    self.assertAlmostEqual(args[0][0]['bandwidth_mbps'], expected_bandwidth_mbps)


  @patch('google.cloud.bigquery.Client')
  def test_table_already_exists(self, mock_bigquery_client):
    """
    Test case where the BigQuery table already exists.
    """
    # Configure the mock client to simulate an existing table
    mock_client_instance = mock_bigquery_client.return_value
    # get_table should not raise NotFound, indicating table exists
    mock_client_instance.get_table.return_value = MagicMock()

    print(f"\n--- Running test_table_already_exists ---")
    result = helper.create_bigquery_table_if_not_exists(
        self.project_id,
        self.dataset_id,
        self.table_id,
        self.schema
    )

    self.assertTrue(result)
    # Verify that get_table was called
    mock_client_instance.get_table.assert_called_once()
    # Verify that create_table was NOT called
    mock_client_instance.create_table.assert_not_called()
    print(f"--- Finished test_table_already_exists ---")

  @patch('google.cloud.bigquery.Client')
  def test_table_does_not_exist_and_is_created(self, mock_bigquery_client):
    """
    Test case where the BigQuery table does not exist and is successfully created.
    """
    # Configure the mock client to simulate a non-existent table
    mock_client_instance = mock_bigquery_client.return_value
    # get_table should raise NotFound, indicating table does not exist
    mock_client_instance.get_table.side_effect = NotFound("Table not found")
    # create_table should return a mock table object upon successful creation
    mock_client_instance.create_table.return_value = MagicMock(table_id=self.table_id)

    print(f"\n--- Running test_table_does_not_exist_and_is_created ---")
    result = helper.create_bigquery_table_if_not_exists(
        self.project_id,
        self.dataset_id,
        self.table_id,
        self.schema
    )

    self.assertTrue(result)
    # Verify that get_table was called
    mock_client_instance.get_table.assert_called_once()
    # Verify that create_table was called
    mock_client_instance.create_table.assert_called_once()
    # You can add more specific assertions about the arguments passed to create_table if needed
    print(f"--- Finished test_table_does_not_exist_and_is_created ---")

  @patch('google.cloud.bigquery.Client')
  def test_create_table_failure(self, mock_bigquery_client):
    """
    Test case where table creation fails after it's determined not to exist.
    """
    mock_client_instance = mock_bigquery_client.return_value
    mock_client_instance.get_table.side_effect = NotFound("Table not found")
    # Simulate an error during table creation
    mock_client_instance.create_table.side_effect = Exception("API error during creation")

    print(f"\n--- Running test_create_table_failure ---")
    result = helper.create_bigquery_table_if_not_exists(
        self.project_id,
        self.dataset_id,
        self.table_id,
        self.schema
    )

    self.assertFalse(result)
    mock_client_instance.get_table.assert_called_once()
    mock_client_instance.create_table.assert_called_once()
    print(f"--- Finished test_create_table_failure ---")

  @patch('google.cloud.bigquery.Client')
  def test_client_initialization_failure(self, mock_bigquery_client):
    """
    Test case where BigQuery client initialization fails.
    """
    # Simulate an error during client initialization
    mock_bigquery_client.side_effect = Exception("Client init error")

    print(f"\n--- Running test_client_initialization_failure ---")
    result = helper.create_bigquery_table_if_not_exists(
        self.project_id,
        self.dataset_id,
        self.table_id,
        self.schema
    )

    self.assertFalse(result)
    # No further calls should be made if client init fails
    mock_bigquery_client.return_value.get_table.assert_not_called()
    mock_bigquery_client.return_value.create_table.assert_not_called()
    print(f"--- Finished test_client_initialization_failure ---")


if __name__ == '__main__':
  unittest.main(argv=['first-arg-is-ignored'], exit=False)
