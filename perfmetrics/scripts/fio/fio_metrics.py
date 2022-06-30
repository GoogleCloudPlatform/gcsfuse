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
from typing import Any, Dict, List

from gsheet import gsheet

JOBNAME = 'jobname'
GLOBAL_OPTS = 'global options'
JOBS = 'jobs'
JOB_OPTS = 'job options'
FILESIZE = 'filesize'
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
IOPS = 'iops'
BW_BYTES = 'bw_bytes'
LAT_NS = 'lat_ns'
LAT_S = 'lat_s'
MIN = 'min'
MAX = 'max'
MEAN = 'mean'
PERCENTILE = 'percentile'
P20 = '20.000000'
P50 = '50.000000'
P90 = '90.000000'
P95 = '95.000000'
LAT_20_PERC = 'lat_20_perc'
LAT_50_PERC = 'lat_50_perc'
LAT_90_PERC = 'lat_90_perc'
LAT_95_PERC = 'lat_95_perc'
IO_BYTES = 'io_bytes'

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

NS_TO_S = 10**(-9)


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
        [{'jobname': '1_thread', 'filesize': 50000, 'num_threads': 40, 'rw': 'read'
        'start_time': 1653027084, 'end_time': 1653027155, 'iops': 95.26093,
        'bw_bytes': 99888128, 'lat_s': {'min': 0.35337776000000004, 'max':
        1.6975198690000002, 'mean': 0.41775487677469203, 'lat_20_perc':
        0.37958451200000004, 'lat_50_perc': 0.38797312, 'lat_90_perc':
        0.49283072000000006, 'lat_95_perc': 0.526385152}, 'io_bytes':
        6040846336}]

    Raises:
      KeyError: Key is missing in the json output
      NoValuesError: Data not present in json object

    """

    if not fio_out:
      raise NoValuesError('No data in json object')

    global_filesize = ''
    global_ramptime_ms = 0
    # default readwrite value
    global_rw = 'read'
    if GLOBAL_OPTS in fio_out:
      if FILESIZE in fio_out[GLOBAL_OPTS]:
        global_filesize = fio_out[GLOBAL_OPTS][FILESIZE]

      if RAMPTIME in fio_out[GLOBAL_OPTS]:
        global_ramptime_ms = _convert_value(fio_out[GLOBAL_OPTS][RAMPTIME],
                                            RAMPTIME_CONVERSION, 's')
      if RW in fio_out[GLOBAL_OPTS]:
        global_rw = fio_out[GLOBAL_OPTS][RW]

    all_jobs = []
    prev_start_time_s = 0
    rev_start_end_times = []
    # getting start and end times of each job
    # Looping from end since the given time is the final end time
    for job in reversed(fio_out[JOBS]):
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
      end_time_ms = prev_start_time_s * 1000 if prev_start_time_s > 0 else fio_out[
          TIMESTAMP_MS]
      # job start time = job end time - job runtime - ramp time
      start_time_ms = end_time_ms - job_rw[RUNTIME] - ramptime_ms

      # converting start and end time to seconds
      start_time_s = start_time_ms // 1000
      end_time_s = round(end_time_ms/1000)
      prev_start_time_s = start_time_s
      rev_start_end_times.append([start_time_s, end_time_s])

    start_end_times = list(reversed(rev_start_end_times))

    for i, job in enumerate(fio_out[JOBS]):
      jobname = ''
      iops = bw_bps = min_lat_s = max_lat_s = mean_lat_s = filesize = 0
      jobname = job[JOBNAME]
      filesize = global_filesize
      ramptime_ms = global_ramptime_ms
      rw = global_rw
      # default value of numjobs
      numjobs = '1'
      if JOB_OPTS in job:
        if NUMJOBS in job[JOB_OPTS]:
          numjobs = job[JOB_OPTS][NUMJOBS]

        if FILESIZE in job[JOB_OPTS]:
          filesize = job[JOB_OPTS][FILESIZE]

        if RW in job[JOB_OPTS]:
          rw = job[JOB_OPTS][RW]

      job_rw = job[_get_rw(rw)]

      iops = job_rw[IOPS]
      bw_bps = job_rw[BW_BYTES]
      min_lat_s = job_rw[LAT_NS][MIN] * NS_TO_S
      max_lat_s = job_rw[LAT_NS][MAX] * NS_TO_S
      mean_lat_s = job_rw[LAT_NS][MEAN] * NS_TO_S
      lat_perc_20 = lat_perc_50 = lat_perc_90 = lat_perc_95 = 0
      if PERCENTILE in job_rw[LAT_NS]:
        lat_perc_20 = job_rw[LAT_NS][PERCENTILE][P20] * NS_TO_S
        lat_perc_50 = job_rw[LAT_NS][PERCENTILE][P50] * NS_TO_S
        lat_perc_90 = job_rw[LAT_NS][PERCENTILE][P90] * NS_TO_S
        lat_perc_95 = job_rw[LAT_NS][PERCENTILE][P95] * NS_TO_S
      io_bytes = job_rw[IO_BYTES]

      start_time_s, end_time_s = start_end_times[i]

      # If jobname and filesize are empty OR start_time=end_time
      # OR all the metrics are zero, log skip warning and continue to next job
      if ((not jobname and not filesize) or (start_time_s == end_time_s) or
          (not iops and not bw_bps and not min_lat_s and not max_lat_s and
           not mean_lat_s)):
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
          IOPS: iops,
          BW_BYTES: bw_bps,
          IO_BYTES: io_bytes,
          LAT_S: {
              MIN: min_lat_s,
              MAX: max_lat_s,
              MEAN: mean_lat_s,
              LAT_20_PERC: lat_perc_20,
              LAT_50_PERC: lat_perc_50,
              LAT_90_PERC: lat_perc_90,
              LAT_95_PERC: lat_perc_95
          }
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
      values.append(
          (job[JOBNAME], job[FILESIZE], job[THREADS], job[RW], job[START_TIME],
           job[END_TIME], job[IOPS], job[BW_BYTES], job[IO_BYTES],
           job[LAT_S][MIN], job[LAT_S][MAX], job[LAT_S][MEAN],
           job[LAT_S][LAT_20_PERC], job[LAT_S][LAT_50_PERC],
           job[LAT_S][LAT_90_PERC], job[LAT_S][LAT_95_PERC]))
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
