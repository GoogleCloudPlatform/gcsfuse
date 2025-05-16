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
  def test_delete_existing_file(self, mock_remove, mock_exists):
    delete_existing_file("/tmp/testfile.bin")

    mock_remove.assert_called_once_with("/tmp/testfile.bin")

  @mock.patch("os.urandom", return_value=b"x" * 1024)
  @mock.patch("builtins.open", new_callable=mock.mock_open)
  def test_write_random_file(self, mock_open_file, mock_urandom):
    file_path = "/tmp/testfile.bin"
    size = 1024

    write_random_file(file_path, size)

    mock_open_file.assert_called_once_with(file_path, 'wb')
    mock_open_file().write.assert_called_once_with(b"x" * 1024)

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

    total = create_files(paths, file_size_in_gb=1e-8)

    self.assertIsNone(total)

if __name__ == '__main__':
  unittest.main(argv=['first-arg-is-ignored'], exit=False)
