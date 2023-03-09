# Copyright 2023 Google Inc. All Rights Reserved.
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

import sys
# For fetching fio_metrics from fio
sys.path.append("./perfmetrics/scripts/")
from fio.fio_metrics import FioMetrics

if __name__ == '__main__':
  argv = sys.argv
  if len(argv) != 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 show_results.py <fio output json filepath>')

  # Fetching metrics from json file
  fio_metrics_obj = FioMetrics()
  data = fio_metrics_obj.get_metrics(argv[1])

  # Fetching results in result.txt file
  file = open("result.txt", "a")
  # Iterating through data
  for d in data :
    # Print filesize only once
    if d['params']['rw'] == "read":
      file.write(str.format(str(round(d["params"]["filesize_kb"]/1024.0,3)) + "MiB" + "\n"))

    # Print Bandwidth
    file.write(str.format(str(round(d["metrics"]["bw_bytes"]/(1024.0*1024.0),2)) + "MiB/s" + "\n"))
  file.close()