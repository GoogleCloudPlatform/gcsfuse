from fio.fio_metrics import FioMetrics

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