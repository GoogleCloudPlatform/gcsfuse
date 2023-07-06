"""Contains constants for fio_metrics.
"""

GLOBAL_OPTS = 'global options'
JOBS = 'jobs'
JOB_OPTS = 'job options'
PARAMS = 'params'
FILESIZE = 'filesize'
FILESIZE_KB = 'filesize_kb'
BS = 'bs'
BS_KB = 'bs_kb'
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

SIZE_TO_KB_CONVERSION = {
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

