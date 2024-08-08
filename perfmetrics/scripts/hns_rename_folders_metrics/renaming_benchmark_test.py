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


if __name__ == '__main__':
  unittest.main()
