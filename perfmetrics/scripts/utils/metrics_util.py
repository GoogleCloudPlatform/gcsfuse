# Copyright 2024 Google Inc. All Rights Reserved.
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

