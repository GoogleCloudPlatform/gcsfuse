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
from unittest import mock
import read_single_thread

class TestReadFiles(unittest.TestCase):

  @mock.patch("builtins.open", new_callable=mock.mock_open, read_data=b"abc")
  @mock.patch("os.path.join", side_effect=lambda a, b: f"{a}/{b}")
  def test_reads_all_files_success(self, mock_join, mock_file):
    total_files = 3
    expected_bytes = 3 * len(b"abc")

    result = read_single_thread.read_all_files(total_files)

    self.assertEqual(result, expected_bytes)
    actual_calls = mock_file.call_args_list
    expected_calls = [
        mock.call(f"{read_single_thread.MOUNT_DIR}/{read_single_thread.FILE_PREFIX}_{i}.bin", "rb")
        for i in range(total_files)
    ]
    self.assertEqual(actual_calls, expected_calls)

  @mock.patch("builtins.open", side_effect=FileNotFoundError("File not found"))
  @mock.patch("os.path.join", side_effect=lambda a, b: f"{a}/{b}")
  def test_file_not_found_raises_runtime_error(self, mock_join, mock_file):
    total_files = 1

    with self.assertRaises(RuntimeError) as cm:
      read_single_thread.read_all_files(total_files)

    self.assertIn("Failed to read file", str(cm.exception))

  @mock.patch("builtins.open", side_effect=PermissionError("Permission denied"))
  @mock.patch("os.path.join", side_effect=lambda a, b: f"{a}/{b}")
  def test_permission_error_raises_runtime_error(self, mock_join, mock_file):
    total_files = 1

    with self.assertRaises(RuntimeError) as cm:
      read_single_thread.read_all_files(total_files)

    self.assertIn("Failed to read file", str(cm.exception))

  @mock.patch("builtins.open", new_callable=mock.mock_open, read_data=b"data")
  @mock.patch("os.path.join", side_effect=lambda a, b: f"{a}/{b}")
  def test_partial_failure(self, mock_join, mock_file):
    # Simulate first file reads correctly, second file throws IOError
    def side_effect(path, mode="rb"):
      if path.endswith("file_0.bin"):
        return mock.mock_open(read_data=b"data").return_value
      else:
        raise IOError("Read error")
    mock_file.side_effect = side_effect

    with self.assertRaises(RuntimeError) as cm:
      read_single_thread.read_all_files(2)

    self.assertIn("Failed to read file", str(cm.exception))
    
  @mock.patch('google.cloud.storage.Client')
  @mock.patch('subprocess.run')
  @mock.patch('os.remove')
  @mock.patch('builtins.print')
  def test_upload_failure(self, mock_print, mock_os_remove, mock_subprocess_run, MockClient):
    """
    Tests the scenario where blob.upload_from_filename fails.
    Verifies that os.remove is still called (cleanup).
    """
    mock_blob = mock.MagicMock()
    mock_blob.exists.return_value = False
    mock_blob.upload_from_filename.side_effect = Exception("GCS upload error") # Simulate upload failure
    mock_bucket = mock.MagicMock()
    mock_bucket.blob.return_value = mock_blob
    MockClient.return_value.get_bucket.return_value = mock_bucket

    # Mock os.path.exists for the finally block in the function
    with mock.patch('os.path.exists', return_value=True):
      bucket_name = "test-bucket"
      total_files = 1
      file_size_gb = 1

      read_single_thread.check_and_create_files(bucket_name, total_files, file_size_gb)

      # Assertions
      mock_subprocess_run.assert_called_once()
      mock_blob.upload_from_filename.assert_called_once()
      mock_os_remove.assert_called_once() # Should be called for cleanup

      mock_print.assert_any_call(mock.ANY) # Check if print was called
      mock_print.assert_any_call(f"Error uploading {read_single_thread.FILE_PREFIX}_{file_size_gb}_0.bin to GCS: GCS upload error")

  @mock.patch("read_single_thread.os.remove")
  @mock.patch("read_single_thread.os.path.exists", return_value=True)
  @mock.patch("read_single_thread.subprocess.run")
  @mock.patch("read_single_thread.storage.Client")
  def test_file_missing_triggers_creation_and_upload(self, mock_storage_client, mock_subprocess_run, mock_path_exists, mock_os_remove):
    mock_bucket = mock.MagicMock()
    mock_blob = mock.MagicMock()
    mock_blob.exists.return_value = False
    mock_blob.size = None
    mock_bucket.blob.return_value = mock_blob
    mock_storage_client.return_value.get_bucket.return_value = mock_bucket

    read_single_thread.check_and_create_files("test-bucket", total_files=1, file_size_gb=1)

    mock_subprocess_run.assert_called_once()  # fallocate
    mock_blob.upload_from_filename.assert_called_once()
    mock_os_remove.assert_called_once()

  @mock.patch("read_single_thread.os.remove")
  @mock.patch("read_single_thread.os.path.exists", return_value=True)
  @mock.patch("read_single_thread.subprocess.run")
  @mock.patch("read_single_thread.storage.Client")
  def test_file_too_small_triggers_upload(self, mock_storage_client, mock_subprocess_run, mock_path_exists, mock_os_remove):
    mock_bucket = mock.MagicMock()
    mock_blob = mock.MagicMock()
    mock_blob.exists.return_value = True
    mock_blob.size = 1 * (10**9) - 200 * (2**20)  # 200 MiB under expected
    mock_bucket.blob.return_value = mock_blob
    mock_storage_client.return_value.get_bucket.return_value = mock_bucket

    read_single_thread.check_and_create_files("test-bucket", total_files=1, file_size_gb=1)

    mock_subprocess_run.assert_called_once()
    mock_blob.upload_from_filename.assert_called_once()
    mock_os_remove.assert_called_once()

if __name__ == '__main__':
  unittest.main(argv=['first-arg-is-ignored'], exit=False) # For running in environments like notebooks
