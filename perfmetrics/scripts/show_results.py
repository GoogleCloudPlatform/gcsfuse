from fio.fio_metrics import FioMetrics
import sys

if __name__ == '__main__':
  argv = sys.argv
  if len(argv) != 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 show_results.py <fio output json filepath>')

  # Fetching metrics from json file
  fio_metrics_obj = FioMetrics()
  data = fio_metrics_obj.get_metrics(argv[1])
  for d in data :
    if d['params']['rw'] == "read":
      print("Filesize: "+ str(round(d["params"]["filesize_kb"]/1024.0,3)) + "MiB")
      print("Read bw: " + str(round(d["metrics"]["bw_bytes"]/(1024.0*1024.0),2)) + "MiB/s")
    if d['params']['rw'] == "write" :
      print("Write bw: " + str(round(d["metrics"]["bw_bytes"]/(1024.0*1024.0),2)) + "MiB/s")
    if d['params']['rw'] == "randread" :
      print("RandRead bw: " + str(round(d["metrics"]["bw_bytes"]/(1024.0*1024.0),2)) + "MiB/s")
    if d['params']['rw'] == "randwrite" :
      print("RandWrite bw: " + str(round(d["metrics"]["bw_bytes"]/(1024.0*1024.0),2)) + "MiB/s")