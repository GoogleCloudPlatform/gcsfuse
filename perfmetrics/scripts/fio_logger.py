"""This script takes the logging directory path as input and 
stores the output result of fio load tests in the directory along with 
removing log files in the directory which are older than last 10 days.
Usage:
python3 fio_logger.py $PATH_TO_LOG_DIR $FIO_OUTPUT_FILE
"""

import os
from datetime import date
import typing
import sys

def log_fio_output(logging_dir: str, output_file: str):
	files = os.listdir(logging_dir)
	files.sort(reverse=True)

	for file in files[9:]:
		# Logging only last 10 fio output files
		# Hence remove older files.
		os.remove(os.path.join(logging_dir,file))
	copied_file = os.path.join(logging_dir, 'output-{}.json'.format(date.today().strftime("%d-%m-%Y")))
	os.system('cp {} {}'.format(output_file, copied_file))

if __name__ == '__main__':
	log_fio_output(sys.argv[1], sys.argv[2])