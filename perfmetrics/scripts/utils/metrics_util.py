""" Script for removing logs older than 10 days

Usage:
python3 metrics_util.py $PATH_TO_LOG_DIR
"""

import os
import typing
import sys


def remove_old_log_files(logging_dir: str, num_files_retain: int):
  files = os.listdir(logging_dir)
  files.sort(reverse=True)

  for file in files[num_files_retain:]:
    # Logging only last 10 fio output files
    # Hence remove older files.
    os.remove(os.path.join(logging_dir, file))


if __name__ == '__main__':
  remove_old_log_files(sys.argv[1], int(sys.argv[2]))

