"""Extracts required metrics from fio output file and writes to google sheet.

   Takes fio output json filepath as command-line input
   Extracts IOPS, Bandwidth and Latency (min, max, mean) from given input file
   and writes the metrics in appropriate columns in a google sheet

   Usage: blaze run :fio_metrics -- <path to fio output json file>

"""

import json
import sys
from typing import Any, Dict, List

JOBNAME = 'jobname'
GLOBAL_OPTS = 'global options'
JOBS = 'jobs'
JOB_OPTS = 'job options'
FILESIZE = 'filesize'
READ = 'read'
IOPS = 'iops'
BW = 'bw'
LAT = 'lat_ns'
MIN = 'min'
MAX = 'max'
MEAN = 'mean'

UNITS = {BW: 'KiB/s', LAT: 'nsec', IOPS: ''}


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
        jobname, filesize, IOPS, Bandwidth and latency (min, max and mean)

    Args:
      fio_out: JSON object representing the fio output

    Returns:
      List of dicts, contains list of jobs and required metrics for each job
      Example return value:
        [{'jobname': '1_thread', 'filesize': '5mb', 'iops': '85.137657',
        'bw': '99137', 'lat_ns': {'min': '365421594', 'max': '38658496964',
        'mean': '23292225875.57558'}}]

    Raises:
      KeyError: Key is missing in the json output
      NoValuesError: Data not present in json object

    """

    if not fio_out:
      raise NoValuesError('No data in json object')

    global_filesize = ''
    if GLOBAL_OPTS in fio_out:
      if FILESIZE in fio_out[GLOBAL_OPTS]:
        global_filesize = fio_out[GLOBAL_OPTS][FILESIZE]

    all_jobs = []
    for i, job in enumerate(fio_out[JOBS]):
      jobname = iops = bw_kibps = min_lat_ns = max_lat_ns = mean_lat_ns = filesize = ''
      jobname = job[JOBNAME]
      job_read = job[READ]
      iops = job_read[IOPS]
      bw_kibps = job_read[BW]
      min_lat_ns = job_read[LAT][MIN]
      max_lat_ns = job_read[LAT][MAX]
      mean_lat_ns = job_read[LAT][MEAN]
      if JOB_OPTS in job:
        if FILESIZE in job[JOB_OPTS]:
          filesize = job[JOB_OPTS][FILESIZE]

      if not filesize:
        filesize = global_filesize

      # If jobname and filesize are empty OR all the metrics are zero,
      # log skip warning and continue to next job
      if ((not jobname and not filesize) or
          (not iops and not bw_kibps and not min_lat_ns and not max_lat_ns and
           not mean_lat_ns)):
        # TODO(ahanadatta): Print statement will be replaced by google logging.
        print(f'No job details or metrics in json, skipping job index {i}')
        continue

      all_jobs.append({
          JOBNAME: jobname,
          FILESIZE: filesize,
          IOPS: iops,
          BW: bw_kibps,
          LAT: {MIN: min_lat_ns, MAX: max_lat_ns, MEAN: mean_lat_ns}
      })

    if not all_jobs:
      raise NoValuesError('No data could be extracted from file')

    return all_jobs

  def _add_to_gsheet(self, jobs):
    """Add the metric values to respective columns in a google sheet.

    Code to be added later
    Args:
      jobs: list of dicts, contains required metrics for each job
    """
    pass

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
                    'blaze run :fio_metrics -- <fio output json filepath>')

  fio_metrics_obj = FioMetrics()
  print(fio_metrics_obj.get_metrics(argv[1], False))
