""" Script for removing logs older than 10 days

Usage:
python3 fio_logger.py $PATH_TO_LOG_DIR $FIO_OUTPUT_FILE
"""

import os
import typing
import sys


def remove_old_log_files(logging_dir: str):
  files = os.listdir(logging_dir)
  files.sort(reverse=True)

  for file in files[10:]:
    # Logging only last 10 fio output files
    # Hence remove older files.
    os.remove(os.path.join(logging_dir, file))


if __name__ == '__main__':
  remove_old_log_files(sys.argv[1])

