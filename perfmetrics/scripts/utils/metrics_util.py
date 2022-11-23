""" Script for removing logs older than last num_files_retain in descending order

Usage:
python3 metrics_util.py $PATH_TO_LOG_DIR $NUM_FILES_RETAIN
"""

import os
import typing
import sys


def remove_old_files(logging_dir: str, num_files_retain: int):
  files = os.listdir(logging_dir)
  files.sort(reverse=True)

  for file in files[num_files_retain:]:
    # Logging only last num_files_retain fio output files
    # Hence remove older files.
    os.remove(os.path.join(logging_dir, file))


if __name__ == '__main__':
  remove_old_files(sys.argv[1], int(sys.argv[2]))

