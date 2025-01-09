# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Executes vm_metrics.py by passing appropriate arguments.

To run the script:
>> python3 populate_vm_metrics.py <start_time> <end_time>
"""
import socket
import sys
import time
import os
from vm_metrics import vm_metrics


INSTANCE = socket.gethostname()
metric_data_name = ['start_time_sec', 'cpu_utilization_peak','cpu_utilization_mean',
                    'network_bandwidth_peak', 'network_bandwidth_mean', 'gcs/ops_latency',
                    'gcs/read_bytes_count', 'gcs/ops_error_count']

if __name__ == '__main__':
  argv = sys.argv
  if len(argv) != 3:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 populate_vm_metrics.py <start_time> <end_time>')

  print('Waiting for 250 seconds for metrics to be updated on VM...')
  # It takes up to 240 seconds for sampled data to be visible on the VM metrics graph
  # So, waiting for 250 seconds to ensure the returned metrics are not empty
  time.sleep(250)

  vm_metrics_obj = vm_metrics.VmMetrics()

  start_time_sec = int(argv[1])
  end_time_sec = int(argv[2])
  period = end_time_sec - start_time_sec
  print(f'Getting VM metrics for ML model')

  vm_metrics_obj.fetch_metrics_and_write_to_google_sheet(start_time_sec, end_time_sec, INSTANCE, period, 'read', 'ml_metrics')

