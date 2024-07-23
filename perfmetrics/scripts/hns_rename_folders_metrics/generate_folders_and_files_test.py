# Copyright 2024 Google Inc. All Rights Reserved.
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
from generate_folders_and_files import check_for_config_file_inconsistency

class TestRenameFolder(unittest.TestCase):
  def test_missing_bucket_name(self):
    config = {}
    result = check_for_config_file_inconsistency(config)
    self.assertEqual(result, 1)

  def test_missing_keys_from_folder(self):
    config = {
        "name": "test_bucket",
        "folders": {}
    }
    result = check_for_config_file_inconsistency(config)
    self.assertEqual(result, 1)

  def test_missing_keys_from_nested_folder(self):
    config = {
        "name": "test_bucket",
        "nested_folders": {}
    }
    result = check_for_config_file_inconsistency(config)
    self.assertEqual(result, 1)

  def test_folders_num_folder_mismatch(self):
    config = {
        "name": "test_bucket",
        "folders": {
            "num_folders": 2,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 10,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }
    result = check_for_config_file_inconsistency(config)
    self.assertEqual(result, 1)

  def test_nested_folders_num_folder_mismatch(self):
    config = {
        "name": "test_bucket",
        "nested_folders": {
            "folder_name": "test_nested_folder",
            "num_folders": 2,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 10,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }
    result = check_for_config_file_inconsistency(config)
    self.assertEqual(result, 1)

  def test_valid_config(self):
    config = {
        "name": "test_bucket",
        "folders": {
            "num_folders": 1,
            "folder_structure": [
                {
                  "name": "test_folder",
                  "num_files": 1,
                  "file_name_prefix": "file",
                  "file_size": "1kb"
                }
            ]
        },
        "nested_folders": {
            "folder_name": "nested",
            "num_folders": 1,
            "folder_structure": [
                {
                    "name": "test_folder",
                    "num_files": 1,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }
    result = check_for_config_file_inconsistency(config)
    self.assertEqual(result, 0)

if __name__ == '__main__':
  unittest.main()
