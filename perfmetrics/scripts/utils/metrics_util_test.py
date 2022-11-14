"""Test for fio output logging."""

import unittest
import os
from metrics_util import remove_old_log_files


class TestFioLogger(unittest.TestCase):

  def test_fio_log_dir_content(self):
    if 'fio_log_test' in os.listdir('.'):
      os.system('rm -r fio_log_test')
    os.mkdir('fio_log_test')
    os.mkdir('fio_log_test/log_dir')
    for i in range(20):
      os.system('touch fio_log_test/log_dir/{}.txt'.format(i))

    remove_old_log_files('fio_log_test/log_dir')

    log_dir_content = os.listdir('fio_log_test/log_dir')

    log_dir_content.sort()
    expected_dir = [
        '18.txt', '19.txt', '2.txt', '3.txt', '4.txt', '5.txt', '6.txt',
        '7.txt', '8.txt', '9.txt'
    ]
    self.assertEqual(log_dir_content[:10], expected_dir)

    # Clean Up
    os.system('rm -r fio_log_test')


if __name__ == '__main__':
  unittest.main()
