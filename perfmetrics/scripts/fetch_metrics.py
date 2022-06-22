"""Executes fio_metrics.py and vm_metrics.py by passing appropriate arguments.
"""
import socket
import sys
import time
from fio import fio_metrics
from vm_metrics import vm_metrics

START_TIME = 'start_time'
END_TIME = 'end_time'
INSTANCE = socket.gethostname()
PERIOD = 120

if __name__ == '__main__':
  argv = sys.argv
  if len(argv) != 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 fetch_metrics.py <fio output json filepath>')

  fio_metrics_obj = fio_metrics.FioMetrics()
  print('Getting fio metrics...')
  temp = fio_metrics_obj.get_metrics(argv[1])
  print('Waiting for 250 seconds for metrics to be updated on VM...')
  time.sleep(250)
  vm_metrics_obj = vm_metrics.VmMetrics()
  for ind, job in enumerate(temp):
    start_time_sec = job[START_TIME]
    end_time_sec = job[END_TIME]
    print(f'Getting VM metrics for job {ind+1}...')
    vm_metrics_obj.fetch_metrics_and_write_to_google_sheet(
        start_time_sec, end_time_sec, INSTANCE, PERIOD)
