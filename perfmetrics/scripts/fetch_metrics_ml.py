"""Executes vm_metrics.py by passing appropriate arguments.

To run the script:
>> python3 fetch_metrics_ml.py <start_time> <end_time>
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
                    'python3 fetch_metrics_ml.py <start_time> <end_time>')

  print('Waiting for 250 seconds for metrics to be updated on VM...')
  # It takes up to 240 seconds for sampled data to be visible on the VM metrics graph
  # So, waiting for 250 seconds to ensure the returned metrics are not empty
  time.sleep(250)

  vm_metrics_obj = vm_metrics.VmMetrics()
  
  start_time_sec = int(argv[1])
  end_time_sec = int(argv[2])
  period = end_time_sec - start_time_sec
  print(f'Getting VM metrics for ML model')

  vm_metrics_obj.fetch_metrics_and_write_to_google_sheet(start_time_sec, end_time_sec, INSTANCE, period, 'read')

