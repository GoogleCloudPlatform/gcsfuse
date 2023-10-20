# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Extracts required metrics from fio output file and writes to google sheet.

   Takes fio output json filepath as command-line input
   Extracts IOPS, Bandwidth and Latency (min, max, mean) from given input file
   and writes the metrics in appropriate columns in a google sheet

   Usage from perfmetrics/scripts folder:
    python3 -m fio.fio_metrics <path to fio output json file>

"""

from dataclasses import dataclass
import json
import re
import sys
from typing import Any, Dict, List, Tuple, Callable

from fio import constants as consts
from gsheet import gsheet

from bigquery import constants
from bigquery import experiments_gcsfuse_bq


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
REQ_JOB_PARAMS.append(JobParam(consts.RW, consts.RW, lambda val: val, 'read'))

REQ_JOB_PARAMS.append(JobParam(consts.THREADS, consts.NUMJOBS,
                               lambda val: int(val), 1))
REQ_JOB_PARAMS.append(
    JobParam(
        consts.FILESIZE_KB, consts.FILESIZE,
        lambda val: _convert_value(val, consts.FILESIZE_TO_KB_CONVERSION), 0))
# append new params here

REQ_JOB_METRICS = []
REQ_JOB_METRICS.append(JobMetric(consts.IOPS, [consts.IOPS], 1))
REQ_JOB_METRICS.append(JobMetric(consts.BW_BYTES, [consts.BW_BYTES], 1))
REQ_JOB_METRICS.append(JobMetric(consts.IO_BYTES, [consts.IO_BYTES], 1))
REQ_JOB_METRICS.append(JobMetric('lat_s_min',
                                 [consts.LAT_NS, consts.MIN], consts.NS_TO_S))
REQ_JOB_METRICS.append(JobMetric('lat_s_max',
                                 [consts.LAT_NS, consts.MAX], consts.NS_TO_S))
REQ_JOB_METRICS.append(JobMetric('lat_s_mean',
                                 [consts.LAT_NS, consts.MEAN], consts.NS_TO_S))
REQ_JOB_METRICS.extend([
    JobMetric('lat_s_perc_20',
              [consts.LAT_NS, consts.PERCENTILE, consts.P20], consts.NS_TO_S),
    JobMetric('lat_s_perc_50',
              [consts.LAT_NS, consts.PERCENTILE, consts.P50], consts.NS_TO_S),
    JobMetric('lat_s_perc_90',
              [consts.LAT_NS, consts.PERCENTILE, consts.P90], consts.NS_TO_S),
    JobMetric('lat_s_perc_95',
              [consts.LAT_NS, consts.PERCENTILE, consts.P95], consts.NS_TO_S)])
# append new metrics here


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

  Raises:
    KeyError: If empty string is passed as value or if unit is present as key in
    conversion_dict
    ValueError: If string has no numerical part

  Ex: For args value = "5s" and conversion_dict=consts.TIME_TO_MS_CONVERSION
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
    rw_value: str, possible values: read/randread/write/randwrite

  Returns:
    str, read/write

  Raises:
    ValueError: If any rw_value other than read/randread/write/randwrite

  """
  if rw_value in ['read', 'randread']:
    return consts.READ
  if rw_value in ['write', 'randwrite']:
    return consts.WRITE
  raise ValueError('Only read/randread/write/randwrite are supported')


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
      job_params: List of dicts, each dict containing parameters of a job

    Returns:
      List of start and end time tuples, one tuple for each job
      Ex: [(1653027014, 1653027084), (1653027084, 1653027155)]

    Raises:
      KeyError: If RW is not present in any dict in job_params

    """
    # Creating a list of just the 'rw' job parameter. Later, we will
    # loop through the jobs from the end, therefore we are creating
    # reversed rw list for easy access
    rw_rev_list = [job_param[consts.RW] for job_param in reversed(job_params)]

    global_ramptime_ms = 0
    global_startdelay_ms = 0
    if consts.GLOBAL_OPTS in out_json:
      if consts.RAMPTIME in out_json[consts.GLOBAL_OPTS]:
        global_ramptime_ms = _convert_value(
            out_json[consts.GLOBAL_OPTS][consts.RAMPTIME],
            consts.TIME_TO_MS_CONVERSION, 's')
      if consts.STARTDELAY in out_json[consts.GLOBAL_OPTS]:
        global_startdelay_ms = _convert_value(
            out_json[consts.GLOBAL_OPTS][consts.STARTDELAY],
            consts.TIME_TO_MS_CONVERSION, 's')

    next_end_time_ms = 0
    rev_start_end_times = []
    # Looping from end since the given time is the final end time
    for i, job in enumerate(list(reversed(out_json[consts.JOBS]))):
      rw = rw_rev_list[i]
      job_rw = job[_get_rw(rw)]
      ramptime_ms = 0
      startdelay_ms = 0
      if consts.JOB_OPTS in job:
        if consts.RAMPTIME in job[consts.JOB_OPTS]:
          ramptime_ms = _convert_value(job[consts.JOB_OPTS][consts.RAMPTIME],
                                       consts.TIME_TO_MS_CONVERSION, 's')

      if ramptime_ms == 0:
        ramptime_ms = global_ramptime_ms
      if startdelay_ms == 0:
        startdelay_ms = global_startdelay_ms

      # for multiple jobs, end time of one job = start time of next job
      end_time_ms = next_end_time_ms if next_end_time_ms > 0 else out_json[
        consts.TIMESTAMP_MS]
      # job start time = job end time - job runtime - ramp time
      start_time_ms = end_time_ms - job_rw[consts.RUNTIME] - ramptime_ms
      next_end_time_ms = start_time_ms - startdelay_ms

      # converting start and end time to seconds
      start_time_s = start_time_ms // 1000
      end_time_s = round(end_time_ms/1000)
      rev_start_end_times.append((start_time_s, end_time_s))

    return list(reversed(rev_start_end_times))

  def _get_job_params(self, out_json):
    """Returns parameter values of each job.

    We'll extract job parameter from 'global options' or 'job options' in the
    JSON using key specified by `json_name`. The parameter will be formatted
    according to function in `format_param`. This formatted value will be stored
    against `name` key. If no parameter is found in the JSON object, the
    `default` value will be used.

    Args:
      out_json : FIO json output

    Returns:
      List of dicts, each dict containing parameters for a job
        Ex: [{'filesize_kb': 50000, 'num_threads': 40, 'rw': 'read'}

    Function working example:
      Ex: out_json = {"global options": {"filesize": "50M", "numjobs": "40"},
                      "jobs":[{"job options": {"numjobs": "10"}}]
                      }
      For REQ_JOB_PARAMS = [
          JobParam(
              name= RW,
              json_name= RW,
              format_param=lambda val: val,
              default = 'read'
          ),
          JobParam(
              name= THREADS,
              json_name= NUMJOBS,
              format_param=lambda val: int(val),
              default = 1
          ),
          JobParam(
              name= FILESIZE_KB,
              json_name= FILESIZE,
              format_param=lambda val: _convert_value(val,
              consts.FILESIZE_TO_KB_CONVERSION),
              default = 0
          )
      ]
      Extracted parameters would be [{RW:'read', THREADS: 10, FILESIZE_KB:
      50000}]


    """
    # Job parameters specified as Global options
    # Each param is formatted according to its format function before storing
    global_params = {}
    if consts.GLOBAL_OPTS in out_json:
      for param in REQ_JOB_PARAMS:
        # If param not present in global options, default value is used
        if param.json_name in out_json[consts.GLOBAL_OPTS]:
          global_params[param.name] = param.format_param(
              out_json[consts.GLOBAL_OPTS][param.json_name])
        else:
          global_params[param.name] = param.default

    # Job parameters specified as job options overwrite global options
    params = []
    for job in out_json[consts.JOBS]:
      curr_job_params = {}
      if consts.JOB_OPTS in job:
        for param in REQ_JOB_PARAMS:
          # If the param is not present in job options, global param is used
          if param.json_name in job[consts.JOB_OPTS]:
            curr_job_params[param.name] = param.format_param(
                job[consts.JOB_OPTS][param.json_name])
          else:
            curr_job_params[param.name] = global_params[param.name]

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
      NoValuesError: Data not present in json object or key in LEVELS is not
      present in FIO output

    """

    if not fio_out:
      raise NoValuesError('No data in json object')

    job_params = self._get_job_params(fio_out)
    start_end_times = self._get_start_end_times(fio_out, job_params)
    all_jobs = []
    # Get the required metrics for every job
    for i, job in enumerate(fio_out[consts.JOBS]):
      rw = job_params[i][consts.RW]
      job_rw = job[_get_rw(rw)]
      job_metrics = {}
      for metric in REQ_JOB_METRICS:
        val = job_rw
        """
        For metric.levels=['lat_ns', 'percentile', '20.000000']
        After 1st iteration, sub = 'lat_ns', val = job_rw['lat_ns']
        After 2nd iteration, sub = 'percentile', val =
        job_rw['lat_ns']['percentile']
        After 3rd iteration, sub = '20.000000', val =
        job_rw['lat_ns']['percentile']['20.000000'] and hence we get the
        required metric value
        """
        for sub in metric.levels:
          if sub in val:
            val = val[sub]
          else:
            val = 0
            raise NoValuesError(
                f'Required metric {sub} not present in json output')

        job_metrics[metric.name] = val * metric.conversion

      start_time_s, end_time_s = start_end_times[i]

      # start_time>=end_time OR all the metrics are zero,
      # log skip warning and continue to next job
      if ((start_time_s >= end_time_s) or
          (all(not value for value in job_metrics.values()))):
        # TODO(ahanadatta): Print statement will be replaced by logging.
        print(f'No job metrics in json, skipping job index {i}')
        continue

      all_jobs.append({
          consts.PARAMS: job_params[i],
          consts.START_TIME: start_time_s,
          consts.END_TIME: end_time_s,
          consts.METRICS: job_metrics
      })

    if not all_jobs:
      raise NoValuesError('No data could be extracted from file')

    return all_jobs

  def get_values_to_upload(self, jobs):
    """Get the metrics values in a list to export to Google Spreadsheet and BigQuery.

    Args:
      jobs: List of dicts, contains required metrics for each job
    Returns:
      list: A 2-d list consisting of metrics values for each job
    """

    values = []
    for job in jobs:
      row = []
      for param_val in job[consts.PARAMS].values():
        row.append(param_val)

      row.append(job[consts.START_TIME])
      row.append(job[consts.END_TIME])
      for metric_val in job[consts.METRICS].values():
        row.append(metric_val)
      values.append(row)
    return values

  def get_metrics(self, filepath) -> List[Dict[str, Any]]:
    """Returns job metrics obtained from given filepath.

    Args:
      filepath (str): Path of the json file to be parsed

    Returns:
      List of dicts, contains list of jobs and required metrics for each job
    """
    fio_out = self._load_file_dict(filepath)
    job_metrics = self._extract_metrics(fio_out)
    return job_metrics

  def upload_metrics_to_bigquery(self, metrics_data, config_id, start_time_build, table_id_bq):
    """Uploads metrics data for load tests to Google Spreadsheets
    Args:
      metrics_data (list): List of metric values for each job
      config_id (str): configuration ID of the experiment
      start_time_build (int): Start time of the build
      table_id_bq (str): ID of table in BigQuery to which metrics data will be uploaded
    """
    bigquery_obj = experiments_gcsfuse_bq.ExperimentsGCSFuseBQ(constants.PROJECT_ID, constants.DATASET_ID)
    bigquery_obj.upload_metrics_to_table(table_id_bq, config_id, start_time_build, metrics_data)

if __name__ == '__main__':
  argv = sys.argv
  if len(argv) != 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 -m fio.fio_metrics <fio output json filepath>')

  fio_metrics_obj = FioMetrics()
  temp = fio_metrics_obj.get_metrics(argv[1])
  print(temp)
