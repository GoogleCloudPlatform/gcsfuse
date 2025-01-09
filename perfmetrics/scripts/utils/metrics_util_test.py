# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Test for fio output logging."""

import unittest
import os
from metrics_util import remove_old_files


class TestMetricsUtil(unittest.TestCase):

  def setUp(self):
    if 'fio_log_test' in os.listdir('.'):
      os.system('rm -r fio_log_test')
    os.mkdir('fio_log_test')
    os.mkdir('fio_log_test/log_dir')

  def tearDown(self):
    # Clean Up
    os.system('rm -r fio_log_test')

  def test_remove_old_files_when_num_retain_files_less_than_dir_files(self):
    for i in range(10):
      os.system('touch fio_log_test/log_dir/{}.txt'.format(i))

    remove_old_files('fio_log_test/log_dir', 5)

    log_dir_content = os.listdir('fio_log_test/log_dir')
    log_dir_content.sort()
    self.assertEqual(log_dir_content, ['{}.txt'.format(x+5) for x in range(5)])

  def test_remove_old_files_when_dir_files_are_zero(self):
    num_files = len(os.listdir('fio_log_test/log_dir'))
    self.assertEqual(num_files, 0)

    remove_old_files('fio_log_test/log_dir', 2)

    log_dir_content = os.listdir('fio_log_test/log_dir')
    self.assertEqual(log_dir_content, [])

  def test_remove_old_files_when_num_retain_files_more_than_dir_files(self):
    for i in range(10):
      os.system('touch fio_log_test/log_dir/{}.txt'.format(i))

    remove_old_files('fio_log_test/log_dir', 12)

    log_dir_content = os.listdir('fio_log_test/log_dir')
    log_dir_content.sort()
    self.assertEqual(log_dir_content, ['{}.txt'.format(x) for x in range(10)])

if __name__ == '__main__':
  unittest.main()
