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
import os
import read

class TestReadAllFiles(unittest.TestCase):
  """
  Unit tests for the read_all_files function.
  Mocks file system interactions to ensure isolated testing.
  """

  @mock.patch('os.path.join', side_effect=os.path.join)
  @mock.patch('builtins.open', new_callable=mock.mock_open)
  def test_read_all_files_success(self, mock_open, mock_join):
    """
    Tests the read_all_files function with successful file reads.
    Mocks file content to verify correct byte calculation.
    """
    num_files = 3
    # Define the content for each mocked file
    mock_file_contents = [
        b"content_one",  # 11 bytes
        b"content_two_long", # 16 bytes
        b"short"         # 5 bytes
    ]
    expected_total_bytes = sum(len(c) for c in mock_file_contents)

    # Configure mock_open to return different content for each call
    # We need to simulate multiple open calls, so we set up side_effect for read()
    # For each call to open(), a new mock file handle is returned.
    # Each mock file handle's .read() method needs to return specific content.

    # Create a list of mock file handles, each with a specific .read() return value
    mock_file_handles = []
    for content in mock_file_contents:
      mock_handle = mock.MagicMock()
      mock_handle.read.return_value = content
      mock_file_handles.append(mock_handle)

    # Configure mock_open's __enter__ method to return the next mock handle
    # This simulates `with open(...) as f:`
    mock_open.side_effect = mock_file_handles

    # Call the function under test
    actual_total_bytes = read.read_all_files(num_files)

    # Verify that open was called for each file
    self.assertEqual(mock_open.call_count, num_files,
                     f"open() should have been called {num_files} times.")

    # Verify the arguments passed to open()
    expected_open_calls = [
        mock.call(os.path.join(read.MOUNT_DIR, f"{read.FILE_PREFIX}_0.bin"), "rb"),
        mock.call(os.path.join(read.MOUNT_DIR, f"{read.FILE_PREFIX}_1.bin"), "rb"),
        mock.call(os.path.join(read.MOUNT_DIR, f"{read.FILE_PREFIX}_2.bin"), "rb"),
    ]
    mock_open.assert_has_calls(expected_open_calls, any_order=False)

    # Verify os.path.join was called correctly
    expected_join_calls = [
        mock.call(read.MOUNT_DIR, f"{read.FILE_PREFIX}_0.bin"),
        mock.call(read.MOUNT_DIR, f"{read.FILE_PREFIX}_1.bin"),
        mock.call(read.MOUNT_DIR, f"{read.FILE_PREFIX}_2.bin"),
    ]
    mock_join.assert_has_calls(expected_join_calls, any_order=False)

  @mock.patch('os.path.join', side_effect=os.path.join)
  @mock.patch('builtins.open', new_callable=mock.mock_open)
  def test_read_all_files_no_files(self, mock_open, mock_join):
    """
    Tests the read_all_files function when no files are specified.
    Should return 0 bytes and not call open().
    """
    num_files = 0
    expected_total_bytes = 0

    actual_total_bytes = read.read_all_files(num_files)

    self.assertEqual(actual_total_bytes, expected_total_bytes,
                     "Total bytes should be 0 when no files are specified.")
    mock_open.assert_not_called()
    mock_join.assert_not_called()

  @mock.patch('os.path.join', side_effect=os.path.join)
  @mock.patch('builtins.open')
  def test_read_all_files_file_not_found(self, mock_open, mock_join):
    """
    Tests the read_all_files function when a FileNotFoundError occurs.
    Ensures the function handles the error gracefully and continues.
    """
    num_files = 2
    # Simulate FileNotFoundError for the first file, and success for the second
    mock_open.side_effect = [
        FileNotFoundError,
        mock.mock_open(read_data=b"successful_read").return_value
    ]
    expected_total_bytes = len(b"successful_read") # Only the second file contributes

    # Capture print output to verify error message (optional but good for robustness)
    with mock.patch('builtins.print') as mock_print:
      actual_total_bytes = read.read_all_files(num_files)

      self.assertEqual(actual_total_bytes, expected_total_bytes,
                       "Total bytes should only include successfully read files.")
      self.assertEqual(mock_open.call_count, num_files,
                       f"open() should have been called {num_files} times.")
      mock_print.assert_called_with(mock.ANY) # Check if print was called
      # You could add more specific checks for the print message if desired
      # e.g., mock_print.assert_called_with("Warning: File not found at /mnt/data/test_file_0.bin")

  @mock.patch('os.path.join', side_effect=os.path.join)
  @mock.patch('builtins.open')
  def test_read_all_files_io_error(self, mock_open, mock_join):
    """
    Tests the read_all_files function when an IOError occurs.
    Ensures the function handles the error gracefully and continues.
    """
    num_files = 2
    # Simulate IOError for the first file, and success for the second
    mock_open.side_effect = [
        IOError("Permission denied"),
        mock.mock_open(read_data=b"another_successful_read").return_value
    ]
    expected_total_bytes = len(b"another_successful_read") # Only the second file contributes

    with mock.patch('builtins.print') as mock_print:
      actual_total_bytes = read.read_all_files(num_files)

      self.assertEqual(actual_total_bytes, expected_total_bytes,
                       "Total bytes should only include successfully read files.")
      self.assertEqual(mock_open.call_count, num_files,
                       f"open() should have been called {num_files} times.")
      mock_print.assert_called_with(mock.ANY) # Check if print was called
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

      read.check_and_create_files(bucket_name, total_files, file_size_gb)

      # Assertions
      mock_subprocess_run.assert_called_once()
      mock_blob.upload_from_filename.assert_called_once()
      mock_os_remove.assert_called_once() # Should be called for cleanup

      mock_print.assert_any_call(mock.ANY) # Check if print was called
      mock_print.assert_any_call(f"Error uploading {read.FILE_PREFIX}_0.bin to GCS: GCS upload error")

  @mock.patch("read.os.remove")
  @mock.patch("read.os.path.exists", return_value=True)
  @mock.patch("read.subprocess.run")
  @mock.patch("read.storage.Client")
  def test_file_missing_triggers_creation_and_upload(
      self, mock_storage_client, mock_subprocess_run, mock_path_exists, mock_os_remove
  ):
    mock_bucket = mock.MagicMock()
    mock_blob = mock.MagicMock()
    mock_blob.exists.return_value = False
    mock_blob.size = None
    mock_bucket.blob.return_value = mock_blob
    mock_storage_client.return_value.get_bucket.return_value = mock_bucket

    read.check_and_create_files("test-bucket", total_files=1, file_size_gb=1)

    mock_subprocess_run.assert_called_once()  # fallocate
    mock_blob.upload_from_filename.assert_called_once()
    mock_os_remove.assert_called_once()

  @mock.patch("read.os.remove")
  @mock.patch("read.os.path.exists", return_value=True)
  @mock.patch("read.subprocess.run")
  @mock.patch("read.storage.Client")
  def test_file_within_tolerance_does_not_trigger_upload(
      self, mock_storage_client, mock_subprocess_run, mock_path_exists, mock_os_remove
  ):
    mock_bucket = mock.MagicMock()
    mock_blob = mock.MagicMock()
    mock_blob.exists.return_value = True
    mock_blob.size = 1 * (10**9) - 5 * (2**20)  # 5 MiB under expected
    mock_bucket.blob.return_value = mock_blob
    mock_storage_client.return_value.get_bucket.return_value = mock_bucket

    read.check_and_create_files("test-bucket", total_files=1, file_size_gb=1)

    mock_subprocess_run.assert_not_called()
    mock_blob.upload_from_filename.assert_not_called()
    mock_os_remove.assert_not_called()

  @mock.patch("read.os.remove")
  @mock.patch("read.os.path.exists", return_value=True)
  @mock.patch("read.subprocess.run")
  @mock.patch("read.storage.Client")
  def test_file_too_small_triggers_upload(
      self, mock_storage_client, mock_subprocess_run, mock_path_exists, mock_os_remove
  ):
    mock_bucket = mock.MagicMock()
    mock_blob = mock.MagicMock()
    mock_blob.exists.return_value = True
    mock_blob.size = 1 * (10**9) - 200 * (2**20)  # 200 MiB under expected
    mock_bucket.blob.return_value = mock_blob
    mock_storage_client.return_value.get_bucket.return_value = mock_bucket

    read.check_and_create_files("test-bucket", total_files=1, file_size_gb=1)

    mock_subprocess_run.assert_called_once()
    mock_blob.upload_from_filename.assert_called_once()
    mock_os_remove.assert_called_once()

if __name__ == '__main__':
  unittest.main(argv=['first-arg-is-ignored'], exit=False) # For running in environments like notebooks
