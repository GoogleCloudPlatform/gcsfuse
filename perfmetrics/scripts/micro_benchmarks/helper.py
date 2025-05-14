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
from google.cloud import bigquery
import subprocess
import helper

# Define the schema for your table
# This matches the schema you provided in your query.
BENCHMARK_SCHEMA = [
    bigquery.SchemaField("timestamp", "TIMESTAMP", mode="NULLABLE"),
    bigquery.SchemaField("duration_seconds", "FLOAT", mode="NULLABLE"),
    bigquery.SchemaField("bandwidth_mbps", "FLOAT", mode="NULLABLE"),
    bigquery.SchemaField("gcsfuse_config", "STRING", mode="NULLABLE"),
    bigquery.SchemaField("workload_type", "STRING", mode="NULLABLE"),
]

# --- Unit Tests ---

class TestBigQueryTableManagement(unittest.TestCase):

  def setUp(self):
    # Common setup for tests
    self.project_id = "your-project-id"
    self.dataset_id = "benchmark_results"
    self.table_id = "gcsfuse_benchmarks"
    self.schema = BENCHMARK_SCHEMA # Use the defined schema

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

class TestGCSFuseUtils(unittest.TestCase):

  @patch('os.makedirs')
  @patch('subprocess.run')
  def test_mount_bucket_success(self, mock_subprocess_run, mock_os_makedirs):
    """
    Test successful mounting of a GCS bucket.
    """
    mount_dir = "/mnt/gcs_test"
    bucket_name = "my-test-bucket"
    flags = "--implicit-dirs"

    print(f"\n--- Running test_mount_bucket_success ---")
    helper.mount_bucket(mount_dir, bucket_name, flags)

    mock_os_makedirs.assert_called_once_with(mount_dir, exist_ok=True)
    expected_cmd = f"gcsfuse {flags} {bucket_name} {mount_dir}"
    mock_subprocess_run.assert_called_once_with(expected_cmd, shell=True, check=True)
    print(f"--- Finished test_mount_bucket_success ---")

  @patch('os.makedirs')
  @patch('subprocess.run', side_effect=subprocess.CalledProcessError(1, 'cmd'))
  def test_mount_bucket_failure(self, mock_subprocess_run, mock_os_makedirs):
    """
    Test failure during mounting of a GCS bucket.
    """
    mount_dir = "/mnt/gcs_test"
    bucket_name = "my-test-bucket"
    flags = "--implicit-dirs"

    print(f"\n--- Running test_mount_bucket_failure ---")
    with self.assertRaises(subprocess.CalledProcessError):
      helper.mount_bucket(mount_dir, bucket_name, flags)

    mock_os_makedirs.assert_called_once_with(mount_dir, exist_ok=True)
    expected_cmd = f"gcsfuse {flags} {bucket_name} {mount_dir}"
    mock_subprocess_run.assert_called_once_with(expected_cmd, shell=True, check=True)
    print(f"--- Finished test_mount_bucket_failure ---")

  @patch('subprocess.run')
  def test_unmount_gcs_directory_success(self, mock_subprocess_run):
    """
    Test successful unmounting of a GCS directory.
    """
    mount_point = "/mnt/gcs_test"

    print(f"\n--- Running test_unmount_gcs_directory_success ---")
    helper.unmount_gcs_directory(mount_point)

    mock_subprocess_run.assert_called_once_with(["fusermount", "-u", mount_point], check=True)
    print(f"--- Finished test_unmount_gcs_directory_success ---")

  @patch('subprocess.run', side_effect=subprocess.CalledProcessError(1, 'cmd'))
  def test_unmount_gcs_directory_failure(self, mock_subprocess_run):
    """
    Test failure during unmounting of a GCS directory.
    """
    mount_point = "/mnt/gcs_test"

    print(f"\n--- Running test_unmount_gcs_directory_failure ---")
    with self.assertRaises(subprocess.CalledProcessError):
      helper.unmount_gcs_directory(mount_point)

    mock_subprocess_run.assert_called_once_with(["fusermount", "-u", mount_point], check=True)
    print(f"--- Finished test_unmount_gcs_directory_failure ---")

class TestBigQueryLogging(unittest.TestCase):

  def setUp(self):
    self.duration_sec = 10.0
    self.total_bytes = 1024 * 1024 * 100 # 100 MiB
    self.gcsfuse_config = "--some-flag"
    self.workload_type = "read"

  @patch('google.cloud.bigquery.Client')
  @patch('pandas.DataFrame')
  @patch('datetime.datetime')
  def test_log_to_bigquery_success(self, mock_datetime, mock_dataframe, mock_bigquery_client):
    """
    Test successful logging of data to BigQuery.
    """
    # Mock datetime.utcnow() to return a fixed time for consistent testing
    mock_now = MagicMock(spec=datetime)
    mock_now.utcnow.return_value = datetime(2023, 1, 1, 12, 0, 0)
    mock_datetime.utcnow = mock_now.utcnow

    # Mock BigQuery Client and its methods
    mock_client_instance = mock_bigquery_client.return_value
    mock_dataset = mock_client_instance.dataset.return_value
    mock_table = mock_dataset.table.return_value
    mock_load_job = MagicMock()
    mock_client_instance.load_table_from_dataframe.return_value = mock_load_job

    # Mock the DataFrame object that pandas.DataFrame will return
    mock_df_instance = mock_dataframe.return_value
    mock_df_instance.__getitem__.side_effect = lambda key: MagicMock() # Allow df['col'] access
    mock_df_instance.__setitem__.side_effect = lambda key, value: None # Allow df['col'] = value
    mock_df_instance.astype.return_value = mock_df_instance # Chaining for astype

    print(f"\n--- Running test_log_to_bigquery_success ---")
    helper.log_to_bigquery(
        self.duration_sec,
        self.total_bytes,
        self.gcsfuse_config,
        self.workload_type
    )

    # Assertions
    mock_bigquery_client.assert_called_once_with(project=PROJECT_ID)
    mock_client_instance.dataset.assert_called_once_with(DATASET_ID)
    mock_dataset.table.assert_called_once_with(TABLE_ID)

    # Verify DataFrame creation
    expected_data = [{
        "timestamp": datetime(2023, 1, 1, 12, 0, 0),
        "duration_seconds": self.duration_sec,
        "bandwidth_mbps": self.total_bytes / self.duration_sec / (1024 * 1024),
        "gcsfuse_config": self.gcsfuse_config,
        "workload_type": self.workload_type,
    }]
    mock_dataframe.assert_called_once()
    # We can't directly assert on the DataFrame content passed to the constructor
    # because it's a new object each time. Instead, we rely on load_table_from_dataframe
    # being called with a DataFrame-like object.

    # Verify load_table_from_dataframe was called with the mock DataFrame and table_ref
    mock_client_instance.load_table_from_dataframe.assert_called_once_with(
        mock_df_instance, # This will be the mock DataFrame instance
        mock_table
    )
    # Verify that .result() was called on the load job
    mock_load_job.result.assert_called_once()
    print(f"--- Finished test_log_to_bigquery_success ---")

  @patch('google.cloud.bigquery.Client')
  @patch('pandas.DataFrame')
  @patch('datetime.datetime')
  def test_log_to_bigquery_failure(self, mock_datetime, mock_dataframe, mock_bigquery_client):
    """
    Test failure during logging of data to BigQuery (e.g., load job fails).
    """
    # Mock datetime.utcnow()
    mock_now = MagicMock(spec=datetime)
    mock_now.utcnow.return_value = datetime(2023, 1, 1, 12, 0, 0)
    mock_datetime.utcnow = mock_now.utcnow

    mock_client_instance = mock_bigquery_client.return_value
    mock_dataset = mock_client_instance.dataset.return_value
    mock_table = mock_dataset.table.return_value
    mock_load_job = MagicMock()
    # Simulate an error when calling .result() on the load job
    mock_load_job.result.side_effect = Exception("BigQuery load job failed")
    mock_client_instance.load_table_from_dataframe.return_value = mock_load_job

    mock_df_instance = mock_dataframe.return_value
    mock_df_instance.__getitem__.side_effect = lambda key: MagicMock()
    mock_df_instance.__setitem__.side_effect = lambda key, value: None
    mock_df_instance.astype.return_value = mock_df_instance

    print(f"\n--- Running test_log_to_bigquery_failure ---")
    with self.assertRaises(Exception): # Expect the exception to be re-raised
      helper.log_to_bigquery(
          self.duration_sec,
          self.total_bytes,
          self.gcsfuse_config,
          self.workload_type
      )

    mock_bigquery_client.assert_called_once_with(project=PROJECT_ID)
    mock_client_instance.load_table_from_dataframe.assert_called_once()
    mock_load_job.result.assert_called_once()
    print(f"--- Finished test_log_to_bigquery_failure ---")


# To run the unit tests, you would typically save this file (e.g., as test_bigquery.py)
# and then run `python -m unittest test_bigquery.py` from your terminal.
# Ensure you have google-cloud-bigquery and pandas installed:
# `pip install google-cloud-bigquery pandas`

# Example of how to call the functions (uncomment to run, requires BigQuery setup and gcsfuse installed)
if __name__ == "__main__":
  # Replace with your actual Google Cloud Project ID
  YOUR_PROJECT_ID = "your-gcp-project-id" # !!! IMPORTANT: Replace with your actual project ID
  YOUR_DATASET_ID = "benchmark_results"
  YOUR_TABLE_ID = "gcsfuse_benchmarks"

  # You can also run specific test classes:
  print("\n--- Running TestBigQueryTableManagement ---")
  suite_table_management = unittest.TestSuite()
  suite_table_management.addTest(unittest.makeSuite(TestBigQueryTableManagement))
  runner_table_management = unittest.TextTestRunner(verbosity=2)
  runner_table_management.run(suite_table_management)

  print("\n--- Running TestGCSFuseUtils ---")
  suite_gcsfuse_utils = unittest.TestSuite()
  suite_gcsfuse_utils.addTest(unittest.makeSuite(TestGCSFuseUtils))
  runner_gcsfuse_utils = unittest.TextTestRunner(verbosity=2)
  runner_gcsfuse_utils.run(suite_gcsfuse_utils)

  print("\n--- Running TestBigQueryLogging ---")
  suite_bigquery_logging = unittest.TestSuite()
  suite_bigquery_logging.addTest(unittest.makeSuite(TestBigQueryLogging))
  runner_bigquery_logging = unittest.TextTestRunner(verbosity=2)
  runner_bigquery_logging.run(suite_bigquery_logging)
