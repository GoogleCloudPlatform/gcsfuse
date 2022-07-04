"""Extracts required metrics from fio output file and writes to google sheet.

   Takes fio output json filepath as command-line input
   Extracts IOPS, Bandwidth and Latency (min, max, mean) from given input file
   and writes the metrics in appropriate columns in a google sheet

   Usage from perfmetrics/scripts folder:
    python3 -m fio.fio_metrics <path to fio output json file>

"""

import json
import re
import sys
from typing import Any, Dict, List, Tuple

from gsheet import gsheet

JOBNAME = 'jobname'
GLOBAL_OPTS = 'global options'
JOBS = 'jobs'
JOB_OPTS = 'job options'
FILESIZE = 'filesize'
FILESIZE_KB = 'filesize_kb'
NUMJOBS = 'numjobs'
THREADS = 'num_threads'
TIMESTAMP_MS = 'timestamp_ms'
RUNTIME = 'runtime'
RAMPTIME = 'ramp_time'
START_TIME = 'start_time'
END_TIME = 'end_time'
RW = 'rw'
READ = 'read'
WRITE = 'write'
METRICS = 'metrics'
IOPS = 'iops'
BW_BYTES = 'bw_bytes'
LAT_NS = 'lat_ns'
MIN = 'min'
MAX = 'max'
MEAN = 'mean'
PERCENTILE = 'percentile'
P20 = '20.000000'
P50 = '50.000000'
P90 = '90.000000'
P95 = '95.000000'
IO_BYTES = 'io_bytes'

NS_TO_S = 10**(-9)

REQ_JOB_PARAMS = [
    {
        'name': JOBNAME,
        'levels': [JOBNAME],
        'format': lambda val: val
    },
    {
        'name': FILESIZE_KB,
        'levels': [JOB_OPTS, FILESIZE],
        'format': lambda val: _convert_value(val, FILESIZE_CONVERSION)
    },
    {
        'name': THREADS,
        'levels': [JOB_OPTS, NUMJOBS],
        'format': lambda val: val
    }
]

REQ_JOB_METRICS = [
    {
        'name': IOPS,
        'levels': [IOPS],
        'conversion': 1
    },
    {
        'name': BW_BYTES,
        'levels': [BW_BYTES],
        'conversion': 1
    },
    {
        'name': IO_BYTES,
        'levels': [IO_BYTES],
        'conversion': 1
    },
    {
        'name': 'lat_s_mean',
        'levels': [LAT_NS, MEAN],
        'conversion': NS_TO_S
    },
    {
        'name': 'lat_s_min',
        'levels': [LAT_NS, MIN],
        'conversion': NS_TO_S
    },
    {
        'name': 'lat_s_max',
        'levels': [LAT_NS, MAX],
        'conversion': NS_TO_S
    },
    {
        'name': 'lat_s_perc_20',
        'levels': [LAT_NS, PERCENTILE, P20],
        'conversion': NS_TO_S
    },
    {
        'name': 'lat_s_perc_50',
        'levels': [LAT_NS, PERCENTILE, P50],
        'conversion': NS_TO_S
    },
    {
        'name': 'lat_s_perc_90',
        'levels': [LAT_NS, PERCENTILE, P90],
        'conversion': NS_TO_S
    },
    {
        'name': 'lat_s_perc_95',
        'levels': [LAT_NS, PERCENTILE, P95],
        'conversion': NS_TO_S
    }
]

# Google sheet worksheet
WORKSHEET_NAME = 'fio_metrics!'

FILESIZE_CONVERSION = {
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

RAMPTIME_CONVERSION = {
    'us': 10**(-3),
    'ms': 1,
    's': 1000,
    'm': 60*1000,
    'h': 3600*1000,
    'd': 24*3600*1000
}


def _convert_value(value, conversion_dict, default_unit=''):
  """Converts data strings to a particular unit based on conversion_dict.

  Args:
    value: String, contains data value[+unit]
    conversion_dict: Dictionary containing units and their respective
      multiplication factor
    default_unit: String, specifies the default unit, used if no unit is present
      in 'value'. Ex: In the job file, we can set ramp_time as "10s" or "10".
      For the latter, the default unit (seconds) is considered.

  Returns:
    Int, number in a specific unit

  Ex: For args value = "5s" and conversion_dict=RAMPTIME_CONVERSION
      "5s" will be converted to 5000 milliseconds and 5000 will be returned

  """
  num_unit = re.findall('[0-9]+|[A-Za-z]+', value)
  if len(num_unit) == 2:
    unit = num_unit[1]
  else:
    unit = default_unit
  num = num_unit[0]
  mult_factor = conversion_dict[unit.lower()]
  converted_num = int(num) * mult_factor
  return converted_num


def _get_rw(rw_value):
  if rw_value in ['read', 'randread']:
    return READ
  if rw_value in ['write', 'randwrite']:
    return WRITE


class NoValuesError(Exception):
  """Some data is missing from the json output file."""


class FioMetrics:
  """Handles logic related to parsing fio output and writing them to google sheet.

  """

  def _load_file_dict(self, filepath) -> Dict[str, Any]:
    """Reads json data from given filepath and returns json object.

    Args:
      filepath : str
        Path of the json file to be parsed

    Returns:
      JSON object, contains json data loaded from given filepath

    Raises:
      OSError: If input filepath doesn't exist
      ValueError: file is not in proper JSON format
      NoValuesError: file doesn't contain JSON data

    """
    fio_out = {}
    f = open(filepath, 'r')
    try:
      fio_out = json.load(f)
    except ValueError as e:
      raise e
    finally:
      f.close()

    if not fio_out:  # Empty JSON object
      raise NoValuesError(f'JSON file {filepath} returned empty object')
    return fio_out

  def _get_start_end_times(self, out_json, global_rw) -> List[Tuple[int]]:
    """Returns start and end times of each job as a list.

    Args:
      out_json : FIO json output
      global_rw: Global read/write value

    Returns:
      List of start and end time tuples, one tuple for each job
      Ex: [(1653027074, 1653027084), (1653027084, 1653027155)]

    """
    global_ramptime_ms = 0
    if GLOBAL_OPTS in out_json:
      if RAMPTIME in out_json[GLOBAL_OPTS]:
        global_ramptime_ms = _convert_value(out_json[GLOBAL_OPTS][RAMPTIME],
                                            RAMPTIME_CONVERSION, 's')

    prev_start_time_s = 0
    rev_start_end_times = []
    # Looping from end since the given time is the final end time
    for job in reversed(out_json[JOBS]):
      ramptime_ms = 0
      rw = global_rw
      if JOB_OPTS in job:
        if RAMPTIME in job[JOB_OPTS]:
          ramptime_ms = _convert_value(job[JOB_OPTS][RAMPTIME],
                                       RAMPTIME_CONVERSION, 's')

        if RW in job[JOB_OPTS]:
          rw = job[JOB_OPTS][RW]

      job_rw = job[_get_rw(rw)]
      if ramptime_ms == 0:
        ramptime_ms = global_ramptime_ms

      # for multiple jobs, end time of one job = start time of next job
      end_time_ms = prev_start_time_s * 1000 if prev_start_time_s > 0 else out_json[
          TIMESTAMP_MS]
      # job start time = job end time - job runtime - ramp time
      start_time_ms = end_time_ms - job_rw[RUNTIME] - ramptime_ms

      # converting start and end time to seconds
      start_time_s = start_time_ms // 1000
      end_time_s = round(end_time_ms/1000)
      prev_start_time_s = start_time_s
      rev_start_end_times.append((start_time_s, end_time_s))

    return list(reversed(rev_start_end_times))

  def _extract_metrics(self, fio_out) -> List[Dict[str, Any]]:
    """Extracts and returns required metrics from fio output dict.

      The extracted metrics are stored in a list. Each entry in the list is a
      dictionary. Each dictionary stores the following fio metrics related
      to a particualar job:
        jobname, filesize, number of threads, IOPS, Bandwidth and latency (min,
        max and mean)

    Args:
      fio_out: JSON object representing the fio output

    Returns:
      List of dicts, contains list of jobs and required metrics for each job
      Example return value:
        [{'jobname': '2_thread', 'filesize': 50000, 'num_threads': 40, 'rw':
        'read', 'start_time': 1653027084, 'end_time': 1653027155, 'metrics':
        {'iops': 95.26093, 'bw_bytes': 99888324, 'io_bytes': 6040846336,
        'lat_s_mean': 0.41775487677469203, 'lat_s_min': 0.35337776000000004,
        'lat_s_max': 1.6975198690000002, 'lat_s_perc_20': 0.37958451200000004,
        'lat_s_perc_50': 0.38797312, 'lat_s_perc_90': 0.49283072000000006,
        'lat_s_perc_95': 0.526385152}}]

    Raises:
      KeyError: Key is missing in the json output
      NoValuesError: Data not present in json object

    """

    if not fio_out:
      raise NoValuesError('No data in json object')

    global_filesize = ''
    # default readwrite value
    global_rw = 'read'
    # default value of numjobs
    global_numjobs = '1'
    if GLOBAL_OPTS in fio_out:
      if FILESIZE in fio_out[GLOBAL_OPTS]:
        global_filesize = fio_out[GLOBAL_OPTS][FILESIZE]

      if RW in fio_out[GLOBAL_OPTS]:
        global_rw = fio_out[GLOBAL_OPTS][RW]

      if NUMJOBS in fio_out[GLOBAL_OPTS]:
        global_numjobs = fio_out[GLOBAL_OPTS][NUMJOBS]

    start_end_times = self._get_start_end_times(fio_out, global_rw)
    all_jobs = []

    for i, job in enumerate(fio_out[JOBS]):
      jobname = ''
      jobname = job[JOBNAME]
      filesize = global_filesize
      rw = global_rw
      numjobs = global_numjobs
      if JOB_OPTS in job:
        if NUMJOBS in job[JOB_OPTS]:
          numjobs = job[JOB_OPTS][NUMJOBS]

        if FILESIZE in job[JOB_OPTS]:
          filesize = job[JOB_OPTS][FILESIZE]

        if RW in job[JOB_OPTS]:
          rw = job[JOB_OPTS][RW]

      job_rw = job[_get_rw(rw)]
      job_metrics = {}
      for metric in REQ_JOB_METRICS:
        val = job_rw
        for sub in metric['levels']:
          if sub in val:
            val = val[sub]
          else:
            val = 0
            break

        job_metrics[metric['name']] = val * metric['conversion']

      start_time_s, end_time_s = start_end_times[i]

      # If jobname and filesize are empty OR start_time=end_time
      # OR all the metrics are zero, log skip warning and continue to next job
      if ((not jobname and not filesize) or (start_time_s == end_time_s) or
          (all(not value for value in job_metrics.values()))):
        # TODO(ahanadatta): Print statement will be replaced by logging.
        print(f'No job details or metrics in json, skipping job index {i}')
        continue

      filesize_kb = _convert_value(filesize, FILESIZE_CONVERSION)
      numjobs = int(numjobs)

      all_jobs.append({
          JOBNAME: jobname,
          FILESIZE: filesize_kb,
          THREADS: numjobs,
          RW: rw,
          START_TIME: start_time_s,
          END_TIME: end_time_s,
          METRICS: job_metrics
      })

    if not all_jobs:
      raise NoValuesError('No data could be extracted from file')

    return all_jobs

  def _add_to_gsheet(self, jobs):
    """Add the metric values to respective columns in a google sheet.

    Args:
      jobs: list of dicts, contains required metrics for each job
    """

    values = []
    for job in jobs:
      row = [
          job[JOBNAME], job[FILESIZE], job[THREADS], job[RW], job[START_TIME],
          job[END_TIME]
      ]
      for metric_val in job[METRICS].values():
        row.append(metric_val)
      values.append(row)

    gsheet.write_to_google_sheet(WORKSHEET_NAME, values)

  def get_metrics(self, filepath, add_to_gsheets=True) -> List[Dict[str, Any]]:
    """Returns job metrics obtained from given filepath and writes to gsheets.

    Args:
      filepath : str
        Path of the json file to be parsed
      add_to_gsheets: bool, optional, default:True
        Whether job metrics should be written to Google sheets or not

    Returns:
      List of dicts, contains list of jobs and required metrics for each job
    """
    fio_out = self._load_file_dict(filepath)
    job_metrics = self._extract_metrics(fio_out)
    if add_to_gsheets:
      self._add_to_gsheet(job_metrics)

    return job_metrics

if __name__ == '__main__':
  argv = sys.argv
  if len(argv) != 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 fio_metrics.py <fio output json filepath>')

  fio_metrics_obj = FioMetrics()
  temp = fio_metrics_obj.get_metrics(argv[1])
  print(temp)

