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

  def test_remove_old_files_less_than_total_files(self):
    for i in range(10):
      os.system('touch fio_log_test/log_dir/{}.txt'.format(i))

    remove_old_files('fio_log_test/log_dir', 5)

    log_dir_content = os.listdir('fio_log_test/log_dir')
    log_dir_content.sort()
    expected_dir = ['{}.txt'.format(x+5) for x in range(5)]
    self.assertEqual(log_dir_content, expected_dir)

  def test_remove_old_files_with_empty_directory(self):
    remove_old_files('fio_log_test/log_dir', 2)

    log_dir_content = os.listdir('fio_log_test/log_dir')
    log_dir_content.sort()
    expected_dir = []
    self.assertEqual(log_dir_content, expected_dir)

  def test_num_retain_files_more_than_dir_files(self):
    for i in range(10):
      os.system('touch fio_log_test/log_dir/{}.txt'.format(i))

    remove_old_files('fio_log_test/log_dir', 12)

    log_dir_content = os.listdir('fio_log_test/log_dir')
    log_dir_content.sort()
    expected_dir = ['{}.txt'.format(x) for x in range(10)]
    self.assertEqual(log_dir_content, expected_dir)

if __name__ == '__main__':
  unittest.main()
