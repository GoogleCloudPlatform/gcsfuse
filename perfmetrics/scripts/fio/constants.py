from dataclasses import dataclass
from typing import Any, Dict, List, Tuple, Callable

GLOBAL_OPTS = 'global options'
JOBS = 'jobs'
JOB_OPTS = 'job options'
PARAMS = 'params'
FILESIZE = 'filesize'
FILESIZE_KB = 'filesize_kb'
NUMJOBS = 'numjobs'
THREADS = 'num_threads'
TIMESTAMP_MS = 'timestamp_ms'
RUNTIME = 'runtime'
RAMPTIME = 'ramp_time'
STARTDELAY = 'startdelay'
START_TIME = 'start_time'
END_TIME = 'end_time'
RW = 'rw'
READ = 'read'
WRITE = 'write'
METRICS = 'metrics'
IOPS = 'iops'
BW_BYTES = 'bw_bytes'
IO_BYTES = 'io_bytes'
LAT_NS = 'lat_ns'
MIN = 'min'
MAX = 'max'
MEAN = 'mean'
PERCENTILE = 'percentile'
P20 = '20.000000'
P50 = '50.000000'
P90 = '90.000000'
P95 = '95.000000'

NS_TO_S = 10**(-9)


@dataclass(frozen=True)
class JobParam:
  """Dataclass for a FIO job parameter.

  name: Can be any suitable value, it refers to the output dictionary key for
  the parameter. To be used when creating parameter dict for each job.
  json_name: Must match the FIO job specification key. Key for parameter inside 
  'global options'/'job options' dictionary
    Ex: For output json = {"global options": {"filesize":"50M"}, "jobs": [
    "job options": {"rw": "read"}]}
    `json_name` for file size will be "filesize" and that for readwrite will be
    "rw"
  format_param: Function returning formatted parameter value. Needed to convert
  parameter to plottable values
    Ex: 'filesize' is obtained as '50M', but we need to convert it to integer
    showing size in kb in order to maintain uniformity
  default: Default value for the parameter

  """
  name: str
  json_name: str
  format_param: Callable[[str], Any]
  default: Any


@dataclass(frozen=True)
class JobMetric:
  """Dataclass for a FIO job metric.

  name: Can be any suitable value, it is used as key for the metric 
  when creating metric dict for each job
  levels: Keys for the metric inside 'read'/'write' dictionary in each job.
  Each value in the list must match the key in the FIO output JSON
    Ex: For job = {'read': {'iops': 123, 'latency': {'min': 0}}}
    levels for IOPS will be ['iops'] and for min latency-> ['latency', 'min']
  conversion: Multiplication factor to convert the metric to the desired unit
    Ex: Extracted latency metrics are in nanoseconds, but we need them in
    seconds for plotting. Hence conversion=10^(-9) for latency metrics.
  """
  name: str
  levels: List[str]
  conversion: float


REQ_JOB_PARAMS = []
# DO NOT remove the below append line
REQ_JOB_PARAMS.append(JobParam(RW, RW, lambda val: val, 'read'))

REQ_JOB_PARAMS.append(JobParam(THREADS, NUMJOBS, lambda val: int(val), 1))
# append new params here

REQ_JOB_METRICS = []
REQ_JOB_METRICS.append(JobMetric(IOPS, [IOPS], 1))
REQ_JOB_METRICS.append(JobMetric(BW_BYTES, [BW_BYTES], 1))
REQ_JOB_METRICS.append(JobMetric(IO_BYTES, [IO_BYTES], 1))
REQ_JOB_METRICS.append(JobMetric('lat_s_min', [LAT_NS, MIN], NS_TO_S))
REQ_JOB_METRICS.append(JobMetric('lat_s_max', [LAT_NS, MAX], NS_TO_S))
REQ_JOB_METRICS.append(JobMetric('lat_s_mean', [LAT_NS, MEAN], NS_TO_S))
REQ_JOB_METRICS.extend([
    JobMetric('lat_s_perc_20', [LAT_NS, PERCENTILE, P20], NS_TO_S),
    JobMetric('lat_s_perc_50', [LAT_NS, PERCENTILE, P50], NS_TO_S),
    JobMetric('lat_s_perc_90', [LAT_NS, PERCENTILE, P90], NS_TO_S),
    JobMetric('lat_s_perc_95', [LAT_NS, PERCENTILE, P95], NS_TO_S)])
# append new metrics here

# Google sheet worksheet
WORKSHEET_NAME = 'fio_metrics!'

FILESIZE_TO_KB_CONVERSION = {
    'b': 0.001,
    'k': 1,
    'kb': 1,
    'm': 10**3,
    'mb': 10**3,
    'g': 10**6,
    'gb': 10**6,
    't': 10**9,
    'tb': 10**9,
    'p': 10**12,
    'pb': 10**12
}

TIME_TO_MS_CONVERSION = {
    'us': 10**(-3),
    'ms': 1,
    's': 1000,
    'm': 60*1000,
    'h': 3600*1000,
    'd': 24*3600*1000
}

