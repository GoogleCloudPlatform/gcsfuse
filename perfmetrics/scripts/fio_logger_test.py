"""Test for fio output logging.
"""

import unittest
import os
from fio_logger import log_fio_output

class TestFioLogger(unittest.TestCase):

	def test_fio_log_dir_content(self):
		if 'fio_log_test' in os.listdir('.'):
			os.system('rm -r fio_log_test')
		os.mkdir('fio_log_test')
		os.mkdir('fio_log_test/log_dir')
		for i in range(10):
			os.system('touch fio_log_test/log_dir/{}.txt'.format(i))
		os.system('touch fio_log_test/output.json')

		log_fio_output('fio_log_test/log_dir', 'fio_log_test/output.json')

		log_dir_content = os.listdir('fio_log_test/log_dir')

		log_dir_content.sort()
		expected_dir = ['{}.txt'.format(i+1) for i in range(9)]
		self.assertEqual(log_dir_content[:9], expected_dir)

		# Clean Up
		os.system('rm -r fio_log_test')

if __name__ == '__main__':
  unittest.main()
