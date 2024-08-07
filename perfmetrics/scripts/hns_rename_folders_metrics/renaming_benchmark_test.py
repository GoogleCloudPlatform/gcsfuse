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
import renaming_benchmark
from mock import patch, call

class TestRenamingBenchmark(unittest.TestCase):

  @patch('subprocess.call')
  @patch('time.time')
  def test_record_time_for_folder_rename(self,mock_time,mock_subprocess):
    mount_point="gcs_bucket"
    folder = {
        'name': "test_folder",
        "num_files": 1,
        "file_name_prefix": "file",
        "file_size": "1kb"
    }
    num_samples=2
    mock_time.side_effect = [1.0, 2.0, 3.0, 4.0]
    expected_time_of_operation=[1.0,1.0]
    expected_subprocess_calls=[call("mv ./gcs_bucket/test_folder ./gcs_bucket/test_folder_renamed",shell=True),
                               call("mv ./gcs_bucket/test_folder_renamed ./gcs_bucket/test_folder",shell=True)]

    time_op=renaming_benchmark._record_time_for_folder_rename(mount_point,folder,num_samples)

    self.assertEqual(time_op,expected_time_of_operation)
    mock_subprocess.assert_has_calls(expected_subprocess_calls)

  @patch('subprocess.call')
  @patch('time.time')
  def test_record_time_of_operation(self,mock_time,mock_subprocess):
    mount_point="gcs_bucket"
    dir = {
        "folders": {
            "folder_structure": [
                {
                    'name': "test_folder1",
                    "num_files": 1,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                },
                {
                    'name': "test_folder2",
                    "num_files": 1,
                    "file_name_prefix": "file",
                    "file_size": "1kb"
                }
            ]
        }
    }

    num_samples=2
    mock_time.side_effect = [1.0, 2.0, 3.0, 4.0,1.0, 2.0, 3.0, 4.0]
    expected_time_of_operation={'test_folder1':[1.0,1.0] ,'test_folder2':[1.0,1.0]}
    expected_subprocess_calls=[call("mv ./gcs_bucket/test_folder1 ./gcs_bucket/test_folder1_renamed",shell=True),
                               call("mv ./gcs_bucket/test_folder1_renamed ./gcs_bucket/test_folder1",shell=True),
                               call("mv ./gcs_bucket/test_folder2 ./gcs_bucket/test_folder2_renamed",shell=True),
                               call("mv ./gcs_bucket/test_folder2_renamed ./gcs_bucket/test_folder2",shell=True),]

    time_op=renaming_benchmark._record_time_of_operation(mount_point,dir,num_samples)

    self.assertEqual(time_op,expected_time_of_operation)
    mock_subprocess.assert_has_calls(expected_subprocess_calls)

  @patch('renaming_benchmark.unmount_gcs_bucket')
  @patch('renaming_benchmark.mount_gcs_bucket')
  @patch('renaming_benchmark._record_time_of_operation')
  @patch('renaming_benchmark.log')
  def test_perform_testing_flat(self, mock_log, mock_record_time_of_operation,
      mock_mount_gcs_bucket, mock_unmount_gcs_bucket):
    dir = {
        "name":"flat_bucket",
        "folders":{
            "num_folders":1,
            "folder_structure":{
                'name': "test_folder",
                "num_files": 1,
                "file_name_prefix": "file",
                "file_size": "1kb"
            }
        }
    }
    test_type = "flat"
    num_samples = 4
    results = {}
    mount_flags = "--implicit-dirs --rename-dir-limit=1000000"
    mock_mount_gcs_bucket.return_value="flat_bucket"
    mock_record_time_of_operation.return_value = {"test_folder": [0.1, 0.2, 0.3, 0.4]}
    expected_results = {"flat": {"test_folder": [0.1, 0.2, 0.3, 0.4]}}


    renaming_benchmark._perform_testing(dir, test_type, num_samples, results)


    self.assertEqual(results, expected_results)
    # Verify calls to other functions.
    mock_mount_gcs_bucket.assert_called_once_with(dir["name"], mount_flags, mock_log)
    mock_record_time_of_operation.assert_called_once_with(mock_mount_gcs_bucket.return_value, dir, num_samples)
    mock_unmount_gcs_bucket.assert_called_once_with(dir["name"], mock_log)
    mock_log.error.assert_not_called()  # No errors should be logged


if __name__ == '__main__':
  unittest.main()
