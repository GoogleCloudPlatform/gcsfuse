"""Executes fio_metrics.py and vm_metrics.py by passing appropriate arguments.
"""
import socket
import sys
import time

from fio import fio_metrics
from gsheet import gsheet
# from vm_metrics import vm_metrics

INSTANCE = socket.gethostname()
PERIOD = 120

# Google sheet worksheets
FIO_WORKSHEET_NAME = 'fio_metrics_expt'
VM_WORKSHEET_NAME = 'vm_metrics'

if __name__ == '__main__':
  argv = sys.argv
  if len(argv) < 2:
    raise TypeError('Missing FIO output JSON filepath.\n'
                    'Usage: '
                    'python3 fetch_metrics.py <1st fio output json filepath> '
                    '<2nd fio output json filepath>...')

  fio_metrics_data = []
  fio_metrics_obj = fio_metrics.FioMetrics()
  vm_metrics_data = []
  for filename in argv[1:]:
    print(f'Getting metrics for file {filename}...')
    
    print('Getting fio metrics...')
    temp = fio_metrics_obj.get_metrics(filename)

    print('Waiting for 250 seconds for metrics to be updated on VM...')
    # It takes up to 240 seconds for sampled data to be visible on the VM 
    # metrics graph. So, waiting for 250 seconds to ensure that the returned 
    # metrics are not empty
    # time.sleep(250)

    # vm_metrics_obj = vm_metrics.VmMetrics()
    # Getting VM metrics for every job
    for ind, job in enumerate(temp):
      start_time_sec = job[fio_metrics.consts.START_TIME]
      end_time_sec = job[fio_metrics.consts.END_TIME]
      rw = job[fio_metrics.consts.PARAMS][fio_metrics.consts.RW]
      print(f'Getting VM metrics for job at index {ind+1}...')
      # metrics_data = vm_metrics_obj.fetch_metrics(start_time_sec, end_time_sec,
      #                                             INSTANCE, PERIOD, rw)

      # Appending metrics data to final array
      fio_metrics_data.append(job)
      # for row in metrics_data:
      #   vm_metrics_data.append(row)

  # Write metrics from all files to google sheet
  print('Writing to Google Sheet')
  fio_metrics_obj._add_to_gsheet(fio_metrics_data, FIO_WORKSHEET_NAME)
  # gsheet.write_to_google_sheet(VM_WORKSHEET_NAME, vm_metrics_data)

