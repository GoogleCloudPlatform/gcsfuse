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
import subprocess
import unittest
import renaming_benchmark
from mock import patch,call,mock_open

class TestMountAndUmount(unittest.TestCase):
  @patch('renaming_benchmark.subprocess.call', return_value=0)
  def test_unmount_gcs_bucket(self, mock_subprocess_call):
    renaming_benchmark._unmount_gcs_bucket('fake_bucket')
    self.assertEqual(mock_subprocess_call.call_count, 2)
    self.assertEqual(mock_subprocess_call.call_args_list[0], call(
        'umount -l fake_bucket', shell=True))
    self.assertEqual(mock_subprocess_call.call_args_list[1], call(
        'rm -rf fake_bucket', shell=True))

  @patch('renaming_benchmark.subprocess.call', return_value=1)
  def test_unmount_gcs_bucket_error(self, mock_subprocess_call):
    renaming_benchmark._unmount_gcs_bucket('fake_bucket')
    self.assertEqual(mock_subprocess_call.call_count, 2)
    self.assertEqual(mock_subprocess_call.call_args_list[0], call(
        'umount -l fake_bucket', shell=True))
    self.assertEqual(
        mock_subprocess_call.call_args_list[1], call('bash', shell=True))

  @patch('renaming_benchmark.subprocess.call', return_value=0)
  def test_mount_gcs_bucket(self, mock_subprocess_call):
    directory_name = renaming_benchmark._mount_gcs_bucket('fake_bucket',
                                                         '--implicit-dirs')
    self.assertEqual(directory_name, 'fake_bucket')
    self.assertEqual(mock_subprocess_call.call_count, 2)
    self.assertEqual(mock_subprocess_call.call_args_list, [
        call('mkdir fake_bucket', shell=True),
        call('gcsfuse --implicit-dirs fake_bucket fake_bucket', shell=True)
    ])

  @patch('renaming_benchmark.subprocess.call', return_value=1)
  def test_mount_gcs_bucket_error(self, mock_subprocess_call):
    renaming_benchmark._mount_gcs_bucket('fake_bucket', '--implicit-dirs')
    self.assertEqual(mock_subprocess_call.call_count, 3)
    self.assertEqual(mock_subprocess_call.call_args_list, [
        call('mkdir fake_bucket', shell=True),
        call('gcsfuse --implicit-dirs fake_bucket fake_bucket', shell=True),
        call('bash', shell=True)
    ])


if __name__ == '__main__':
  unittest.main()
