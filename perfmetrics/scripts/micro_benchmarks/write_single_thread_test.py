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
from write_single_thread import  create_files, delete_existing_file, write_random_file

class TestWriteFiles(unittest.TestCase):

  @mock.patch("os.path.exists", return_value=True)
  @mock.patch("os.remove")
  def test_delete_existing_file_success(self, mock_remove, mock_exists):
    result = delete_existing_file("/fake/path")

    self.assertTrue(result)
    mock_remove.assert_called_once_with("/fake/path")

  @mock.patch("os.path.exists", return_value=True)
  @mock.patch("os.remove", side_effect=OSError("Permission denied"))
  def test_delete_existing_file_failure(self, mock_remove, mock_exists):
    result = delete_existing_file("/fake/path")

    self.assertFalse(result)
    mock_remove.assert_called_once_with("/fake/path")

  @mock.patch("os.path.exists", return_value=False)
  @mock.patch("os.remove")
  def test_delete_existing_file_file_not_exist(self, mock_remove, mock_exists):
    # When file doesn't exist, should return True and not call remove
    result = delete_existing_file("/fake/path")

    self.assertTrue(result)
    mock_remove.assert_not_called()

  @mock.patch("builtins.open", new_callable=mock.mock_open)
  @mock.patch("os.urandom", return_value=b'x' * 10)
  def test_write_random_file_success(self, mock_urandom, mock_open):
    result = write_random_file("/fake/file", 10)

    self.assertTrue(result)
    mock_open.assert_called_once_with("/fake/file", "wb")
    mock_urandom.assert_called_once_with(10)

  @mock.patch("builtins.open", side_effect=IOError("Disk full"))
  def test_write_random_file_failure(self, mock_open):
    result = write_random_file("/fake/file", 10)

    self.assertFalse(result)
    mock_open.assert_called_once_with("/fake/file", "wb")

  @mock.patch("os.urandom", return_value=b"x" * 10)
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_create_files_success(self, mock_open_file, mock_urandom):
    paths = ["/tmp/file1.bin", "/tmp/file2.bin"]
    expected_total = 20  # 2 files * 10 bytes

    total = create_files(paths, file_size_in_gb=1e-8)  # ~10 bytes each

    self.assertEqual(total, expected_total)
    self.assertEqual(mock_open_file.call_count, 2)

  @mock.patch("builtins.open", side_effect=Exception("write error"))
  def test_create_files_failure(self, mock_open_file):
    paths = ["/tmp/file1.bin"]

    with self.assertRaises(SystemExit) as cm:
      create_files(paths, file_size_in_gb=1e-8)

    self.assertEqual(cm.exception.code, 1)


if __name__ == '__main__':
  unittest.main(argv=['first-arg-is-ignored'], exit=False)
