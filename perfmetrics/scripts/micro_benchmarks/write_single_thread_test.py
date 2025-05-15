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
from write_single_thread import  create_files

class TestWriteFiles(unittest.TestCase):
  @mock.patch("write_single_thread.os.urandom", return_value=b"x" * 100)
  @mock.patch("write_single_thread.open", new_callable=mock.mock_open)
  @mock.patch("write_single_thread.os.remove")
  @mock.patch("write_single_thread.os.path.exists", return_value=False)
  def test_create_files_success(self, mock_exists, mock_remove, mock_open, mock_urandom):
    result = create_files(2, 1e-7, file_prefix="test")  # Small size to avoid memory issues

    self.assertIsInstance(result, int)
    self.assertGreater(result, 0)
    self.assertEqual(mock_open.call_count, 2)
    mock_remove.assert_not_called()  # Because os.path.exists returned False

  @mock.patch("write_single_thread.os.urandom", return_value=b"x" * 100)
  @mock.patch("write_single_thread.open", new_callable=mock.mock_open)
  @mock.patch("write_single_thread.os.remove", side_effect=PermissionError("Cannot remove"))
  @mock.patch("write_single_thread.os.path.exists", return_value=True)
  def test_create_files_remove_failure(self, mock_exists, mock_remove, mock_open, mock_urandom):
    result = create_files(1, 1e-7, file_prefix="test")

    self.assertIsNone(result)
    mock_remove.assert_called_once()

  @mock.patch("write_single_thread.os.urandom", return_value=b"x" * 100)
  @mock.patch("write_single_thread.open", side_effect=OSError("Disk full"))
  @mock.patch("write_single_thread.os.remove")
  @mock.patch("write_single_thread.os.path.exists", return_value=False)
  def test_create_files_write_failure(self, mock_exists, mock_remove, mock_open, mock_urandom):
    result = create_files(1, 1e-7, file_prefix="test")

    self.assertIsNone(result)
    mock_open.assert_called_once()

if __name__ == '__main__':
  unittest.main(argv=['first-arg-is-ignored'], exit=False)
