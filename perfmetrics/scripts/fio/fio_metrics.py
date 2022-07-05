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

NAME = 'name'
JSON_NAME = 'json_name'
FORMAT = 'format'
DEFAULT = 'default'
LEVELS = 'levels'
CONVERSION = 'conversion'
NS_TO_S = 10**(-9)

""" REQ_JOB_PARAMS:
NAME: key for the parameter, to be used when creating parameter dict for each
Job
JSON_NAME: Key for parameter inside 'global options'/'job options' dictionary
  Ex: For output json = {"global options": {"filesize":"50M"}, "jobs": [
  "job options": {"rw": "read"}]}
  JSON_NAME for filesize will be "filesize" and that for readwrite will be "rw"
FORMAT: Function returning formatted parameter value. Needed to convert
parameter to plottable values
  Ex: 'filesize' is obtained as '50M', but we need to convert it to integer
  showing size in kb in order to maintain uniformity
DEFAULT: Default value for the parameter

Ex: output json = {"global options": {"filesize":"50M"}}
For REQ_JOB_PARAMS = [
    {
        NAME: FILESIZE_KB,
        JSON_NAME: FILESIZE,
        FORMAT: lambda val: _convert_value(val, FILESIZE_CONVERSION),
        DEFAULT: 0
    },
    {
        NAME: RW,
        JSON_NAME: RW,
        FORMAT: lambda val: val,
        DEFAULT: 'read'
    }
]
Extracted parameters would be {FILESIZE_KB: 50000, RW: 'read'}
"""
REQ_JOB_PARAMS = [
    {
        NAME: FILESIZE_KB,
        JSON_NAME: FILESIZE,
        FORMAT: lambda val: _convert_value(val, FILESIZE_CONVERSION),
        DEFAULT: 0
    },
    {
        NAME: THREADS,
        JSON_NAME: NUMJOBS,
        FORMAT: lambda val: int(val),
        DEFAULT: 1
    },
    # Don't remove the below parameter
    {
        NAME: RW,
        JSON_NAME: RW,
        FORMAT: lambda val: val,
        DEFAULT: 'read'
    }
]

""" REQ_JOB_METRICS:
NAME: key for the metric, to be used when creating metric dict for each job
LEVELS: Keys for the metric inside 'read'/'write' dictionary in each job
  Ex: For job = {'read': {'iops':123, 'latency':{'min':0}}}
  LEVELS for IOPS will be ['iops'] and for min latency-> ['latency', 'min']
CONVERSION: Multiplication factor to convert the metric to the desired unit
  Ex: Extracted latency metrics are in nanoseconds, but we need them in seconds
  for plotting. Hence CONVERSION=10^(-9) for latency metrics.
"""
REQ_JOB_METRICS = [
    {
        NAME: IOPS,
        LEVELS: [IOPS],
        CONVERSION: 1
    },
    {
        NAME: BW_BYTES,
        LEVELS: [BW_BYTES],
        CONVERSION: 1
    },
    {
        NAME: IO_BYTES,
        LEVELS: [IO_BYTES],
        CONVERSION: 1
    },
    {
        NAME: 'lat_s_mean',
        LEVELS: [LAT_NS, MEAN],
        CONVERSION: NS_TO_S
    },
    {
        NAME: 'lat_s_min',
        LEVELS: [LAT_NS, MIN],
        CONVERSION: NS_TO_S
    },
    {
        NAME: 'lat_s_max',
        LEVELS: [LAT_NS, MAX],
        CONVERSION: NS_TO_S
    },
    {
        NAME: 'lat_s_perc_20',
        LEVELS: [LAT_NS, PERCENTILE, P20],
        CONVERSION: NS_TO_S
    },
    {
        NAME: 'lat_s_perc_50',
        LEVELS: [LAT_NS, PERCENTILE, P50],
        CONVERSION: NS_TO_S
    },
    {
        NAME: 'lat_s_perc_90',
        LEVELS: [LAT_NS, PERCENTILE, P90],
        CONVERSION: NS_TO_S
    },
    {
        NAME: 'lat_s_perc_95',
        LEVELS: [LAT_NS, PERCENTILE, P95],
        CONVERSION: NS_TO_S
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
  """Converting read/randread/write/randwrite to just read/write.

  Args:
    rw_value: str, 'read'/'randread'/'write'/'randwrite'

  Returns:
    str, 'read'/'write'

  """
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

  def _get_start_end_times(self, out_json, job_params) -> List[Tuple[int]]:
    """Returns start and end times of each job as a list.

    Args:
      out_json : FIO json output
      job_params: List of dicts, Parameters of all jobs

    Returns:
      List of start and end time tuples, one tuple for each job
      Ex: [(1653027014, 1653027084), (1653027084, 1653027155)]

    """
    rw_rev_list = [job_param[RW] for job_param in reversed(job_params)]
    global_ramptime_ms = 0
    if GLOBAL_OPTS in out_json:
      if RAMPTIME in out_json[GLOBAL_OPTS]:
        global_ramptime_ms = _convert_value(out_json[GLOBAL_OPTS][RAMPTIME],
                                            RAMPTIME_CONVERSION, 's')

    prev_start_time_s = 0
    rev_start_end_times = []
    # Looping from end since the given time is the final end time
    for i, job in enumerate(list(reversed(out_json[JOBS]))):
      rw = rw_rev_list[i]
      job_rw = job[_get_rw(rw)]
      ramptime_ms = 0
      if JOB_OPTS in job:
        if RAMPTIME in job[JOB_OPTS]:
          ramptime_ms = _convert_value(job[JOB_OPTS][RAMPTIME],
                                       RAMPTIME_CONVERSION, 's')
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

  def _get_job_params(self, out_json):
    """Returns parameter values of each job.

    Args:
      out_json : FIO json output

    Returns:
      List of dicts, each dict containing parameters for a job
        Ex: [{'filesize_kb': 50000, 'num_threads': 40, 'rw': 'read'}

    Function working example:
      Ex: out_json = {"global options": {"filesize":"50M", "numjobs":"40"},
      "jobs":[{"job options": {"numjobs":"10"}}]}
      For REQ_JOB_PARAMS = [
          {
              NAME: FILESIZE_KB,
              JSON_NAME: FILESIZE,
              FORMAT: lambda val: _convert_value(val, FILESIZE_CONVERSION),
              DEFAULT: 0
          },
          {
              NAME: THREADS,
              JSON_NAME: NUMJOBS,
              FORMAT: lambda val: int(val),
              DEFAULT: 1
          },
          {
              NAME: RW,
              JSON_NAME: RW,
              FORMAT: lambda val: val,
              DEFAULT: 'read'
          }
      ]
      Extracted parameters would be [{FILESIZE_KB: 50000, THREADS: 10, RW:
      'read'}]


    """
    # Job parameters specified as Global options
    # Each param is formatted according to its format function before storing
    global_params = {}
    if GLOBAL_OPTS in out_json:
      for param in REQ_JOB_PARAMS:
        # If param not present in global options, default value is used
        if param[JSON_NAME] in out_json[GLOBAL_OPTS]:
          global_params[param[NAME]] = param[FORMAT](
              out_json[GLOBAL_OPTS][param[JSON_NAME]])
        else:
          global_params[param[NAME]] = param[DEFAULT]

    # Job parameters specified as job options overwrite global options
    params = []
    for job in out_json[JOBS]:
      curr_job_params = {}
      if JOB_OPTS in job:
        for param in REQ_JOB_PARAMS:
          # If the param is not present in job options, global param is used
          if param[JSON_NAME] in job[JOB_OPTS]:
            curr_job_params[param[NAME]] = param[FORMAT](
                job[JOB_OPTS][param[JSON_NAME]])
          else:
            curr_job_params[param[NAME]] = global_params[param[NAME]]

      params.append(curr_job_params)

    return params

  def _extract_metrics(self, fio_out) -> List[Dict[str, Any]]:
    """Extracts and returns required metrics from fio output dict.

      The extracted metrics are stored in a list. Each entry in the list is a
      dictionary. Each dictionary stores the following fio metrics related
      to a particualar job:
        filesize, number of threads, IOPS, Bandwidth and latency (min,
        max and mean)

    Args:
      fio_out: JSON object representing the fio output

    Returns:
      List of dicts, contains list of jobs and required parameters and metrics
      for each job
      Example return value:
        [{'params': {'filesize': 50000, 'num_threads': 40, 'rw': 'read'},
          'start_time': 1653027084, 'end_time': 1653027155, 'metrics':
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

    job_params = self._get_job_params(fio_out)
    start_end_times = self._get_start_end_times(fio_out, job_params)
    all_jobs = []

    # Get the required metrics for every job
    for i, job in enumerate(fio_out[JOBS]):
      rw = job_params[i][RW]
      job_rw = job[_get_rw(rw)]
      job_metrics = {}
      for metric in REQ_JOB_METRICS:
        val = job_rw
        """
        For metric[LEVELS]=['lat_ns', 'percentile', '20.000000']
        After 1st iteration, sub = 'lat_ns', val = job_rw['lat_ns']
        After 2nd iteration, sub = 'percentile', val =
        job_rw['lat_ns']['percentile']
        After 3rd iteration, sub = '20.000000', val =
        job_rw['lat_ns']['percentile']['20.000000'] and hence we get the
        required metric value
        """
        for sub in metric[LEVELS]:
          if sub in val:
            val = val[sub]
          else:
            val = 0
            raise NoValuesError(
                f'Required metric {sub} not present in json output')

        job_metrics[metric[NAME]] = val * metric[CONVERSION]

      start_time_s, end_time_s = start_end_times[i]

      # start_time=end_time OR all the metrics are zero, 
      # log skip warning and continue to next job
      if ((start_time_s == end_time_s) or
          (all(not value for value in job_metrics.values()))):
        # TODO(ahanadatta): Print statement will be replaced by logging.
        print(f'No job metrics in json, skipping job index {i}')
        continue

      all_jobs.append({
          PARAMS: job_params[i],
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
      row = []
      for param_val in job[PARAMS].values():
        row.append(param_val)

      row.append(job[START_TIME])
      row.append(job[END_TIME])
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

