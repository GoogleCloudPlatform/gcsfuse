from absl import flags

from google3.experimental.users.swethv.gcsfuse.dashboard.fio import fio_metrics
from google3.pyglib import resources
from google3.testing.pybase import googletest

FLAGS = flags.FLAGS
TEST_PATH = 'google3/experimental/users/swethv/gcsfuse/dashboard/fio/testdata/'
GOOD_FILE = 'good_out_job.json'
EMPTY_FILE = 'empty_file.json'
EMPTY_JSON_FILE = 'empty_json.json'
PARTIAL_FILE = 'partial_json.json'
NO_METRICS_FILE = 'no_metrics.json'
BAD_FORMAT_FILE = 'bad_format.json'
MULTIPLE_JOBS_GLOBAL_FSIZE_FILE = 'multiple_jobs_global_fsize.json'
MULTIPLE_JOBS_JOB_FSIZE_FILE = 'multiple_jobs_job_fsize.json'


def get_full_filepath(filename):
  filepath = '{}{}'.format(TEST_PATH, filename)
  return resources.GetResourceFilename(filepath)


class FioMetricsTest(googletest.TestCase):

  def setUp(self):
    super().setUp()
    self.fio_metrics_obj = fio_metrics.FioMetrics()

  def test_load_file_dict_good_file(self):
    expected_json = {
        'fio version':
            'fio-3.30',
        'timestamp':
            1653027155,
        'timestamp_ms':
            1653027155355,
        'time':
            'Fri May 20 06:12:35 2022',
        'global options': {
            'direct': '1',
            'fadvise_hint': '0',
            'verify': '0',
            'rw': 'read',
            'bs': '1M',
            'iodepth': '64',
            'invalidate': '1',
            'ramp_time': '10s',
            'runtime': '60s',
            'time_based': '1',
            'nrfiles': '1',
            'thread': '1',
            'filesize': '50M',
            'openfiles': '1',
            'group_reporting': '1',
            'allrandrepeat': '1',
            'directory': 'gcs/50mb',
            'filename_format': '$jobname.$jobnum.$filenum'
        },
        'jobs': [{
            'jobname': '1_thread',
            'groupid': 0,
            'error': 0,
            'eta': 0,
            'elapsed': 80,
            'job options': {
                'numjobs': '40'
            },
            'read': {
                'io_bytes': 6040846336,
                'io_kbytes': 5899264,
                'bw_bytes': 99888324,
                'bw': 97547,
                'iops': 95.26093,
                'runtime': 60476,
                'total_ios': 5761,
                'short_ios': 0,
                'drop_ios': 0,
                'slat_ns': {
                    'min': 0,
                    'max': 0,
                    'mean': 0.0,
                    'stddev': 0.0,
                    'N': 0
                },
                'clat_ns': {
                    'min': 353376970,
                    'max': 1697518879,
                    'mean': 417753956.415726,
                    'stddev': 119951981.880844,
                    'N': 5761,
                    'percentile': {
                        '1.000000': 375390208,
                        '5.000000': 379584512,
                        '10.000000': 379584512,
                        '20.000000': 379584512,
                        '30.000000': 383778816,
                        '40.000000': 383778816,
                        '50.000000': 387973120,
                        '60.000000': 387973120,
                        '70.000000': 396361728,
                        '80.000000': 408944640,
                        '90.000000': 492830720,
                        '95.000000': 526385152,
                        '99.000000': 893386752,
                        '99.500000': 1568669696,
                        '99.900000': 1635778560,
                        '99.950000': 1652555776,
                        '99.990000': 1702887424
                    }
                },
                'lat_ns': {
                    'min': 353377760,
                    'max': 1697519869,
                    'mean': 417754876.774692,
                    'stddev': 119951962.892831,
                    'N': 5761
                },
                'bw_min': 77907,
                'bw_max': 163976,
                'bw_agg': 100.0,
                'bw_mean': 101253.107555,
                'bw_dev': 870.557782,
                'bw_samples': 4614,
                'iops_min': 40,
                'iops_max': 160,
                'iops_mean': 93.168535,
                'iops_stddev': 0.920229,
                'iops_samples': 4614
            },
            'write': {
                'io_bytes': 0,
                'io_kbytes': 0,
                'bw_bytes': 0,
                'bw': 0,
                'iops': 0.0,
                'runtime': 0,
                'total_ios': 0,
                'short_ios': 0,
                'drop_ios': 0,
                'slat_ns': {
                    'min': 0,
                    'max': 0,
                    'mean': 0.0,
                    'stddev': 0.0,
                    'N': 0
                },
                'clat_ns': {
                    'min': 0,
                    'max': 0,
                    'mean': 0.0,
                    'stddev': 0.0,
                    'N': 0
                },
                'lat_ns': {
                    'min': 0,
                    'max': 0,
                    'mean': 0.0,
                    'stddev': 0.0,
                    'N': 0
                },
                'bw_min': 0,
                'bw_max': 0,
                'bw_agg': 0.0,
                'bw_mean': 0.0,
                'bw_dev': 0.0,
                'bw_samples': 0,
                'iops_min': 0,
                'iops_max': 0,
                'iops_mean': 0.0,
                'iops_stddev': 0.0,
                'iops_samples': 0
            },
            'trim': {
                'io_bytes': 0,
                'io_kbytes': 0,
                'bw_bytes': 0,
                'bw': 0,
                'iops': 0.0,
                'runtime': 0,
                'total_ios': 0,
                'short_ios': 0,
                'drop_ios': 0,
                'slat_ns': {
                    'min': 0,
                    'max': 0,
                    'mean': 0.0,
                    'stddev': 0.0,
                    'N': 0
                },
                'clat_ns': {
                    'min': 0,
                    'max': 0,
                    'mean': 0.0,
                    'stddev': 0.0,
                    'N': 0
                },
                'lat_ns': {
                    'min': 0,
                    'max': 0,
                    'mean': 0.0,
                    'stddev': 0.0,
                    'N': 0
                },
                'bw_min': 0,
                'bw_max': 0,
                'bw_agg': 0.0,
                'bw_mean': 0.0,
                'bw_dev': 0.0,
                'bw_samples': 0,
                'iops_min': 0,
                'iops_max': 0,
                'iops_mean': 0.0,
                'iops_stddev': 0.0,
                'iops_samples': 0
            },
            'sync': {
                'total_ios': 0,
                'lat_ns': {
                    'min': 0,
                    'max': 0,
                    'mean': 0.0,
                    'stddev': 0.0,
                    'N': 0
                }
            },
            'job_runtime': 2406719,
            'usr_cpu': 0.004072,
            'sys_cpu': 0.022313,
            'ctx': 5836,
            'majf': 0,
            'minf': 0,
            'iodepth_level': {
                '1': 100.0,
                '2': 0.0,
                '4': 0.0,
                '8': 0.0,
                '16': 0.0,
                '32': 0.0,
                '>=64': 0.0
            },
            'iodepth_submit': {
                '0': 0.0,
                '4': 100.0,
                '8': 0.0,
                '16': 0.0,
                '32': 0.0,
                '64': 0.0,
                '>=64': 0.0
            },
            'iodepth_complete': {
                '0': 0.0,
                '4': 100.0,
                '8': 0.0,
                '16': 0.0,
                '32': 0.0,
                '64': 0.0,
                '>=64': 0.0
            },
            'latency_ns': {
                '2': 0.0,
                '4': 0.0,
                '10': 0.0,
                '20': 0.0,
                '50': 0.0,
                '100': 0.0,
                '250': 0.0,
                '500': 0.0,
                '750': 0.0,
                '1000': 0.0
            },
            'latency_us': {
                '2': 0.0,
                '4': 0.0,
                '10': 0.0,
                '20': 0.0,
                '50': 0.0,
                '100': 0.0,
                '250': 0.0,
                '500': 0.0,
                '750': 0.0,
                '1000': 0.0
            },
            'latency_ms': {
                '2': 0.0,
                '4': 0.0,
                '10': 0.0,
                '20': 0.0,
                '50': 0.0,
                '100': 0.0,
                '250': 0.0,
                '500': 91.355667,
                '750': 6.561361,
                '1000': 1.336574,
                '2000': 0.746398,
                '>=2000': 0.0
            },
            'latency_depth': 64,
            'latency_target': 0,
            'latency_percentile': 100.0,
            'latency_window': 0
        }]
    }

    json_obj = self.fio_metrics_obj._load_file_dict(
        get_full_filepath(GOOD_FILE))

    self.assertEqual(expected_json, json_obj)

  def test_load_non_existent_file_raises_os_error(self):
    json_obj = None

    with self.assertRaises(OSError):
      json_obj = self.fio_metrics_obj._load_file_dict('i_dont_exist')
    self.assertIsNone(json_obj)

  def test_load_file_dict_empty_file_raises_value_error(self):
    json_obj = None

    with self.assertRaises(ValueError):
      self.json_obj = self.fio_metrics_obj._load_file_dict(
          get_full_filepath(EMPTY_FILE))
    self.assertIsNone(json_obj)

  def test_load_file_dict_empty_json_raises_no_values_error(self):
    json_obj = None

    with self.assertRaises(fio_metrics.NoValuesError):
      json_obj = self.fio_metrics_obj._load_file_dict(
          get_full_filepath(EMPTY_JSON_FILE))
    self.assertIsNone(json_obj)

  def test_load_file_dict_bad_format_file_raises_value_error(self):
    """input file is not in JSON format.

    """
    json_obj = None

    with self.assertRaises(ValueError):
      json_obj = self.fio_metrics_obj._load_file_dict(
          get_full_filepath(BAD_FORMAT_FILE))
    self.assertIsNone(json_obj)

  def test_extract_metrics_from_good_file(self):
    json_obj = self.fio_metrics_obj._load_file_dict(
        get_full_filepath(GOOD_FILE))
    expected_metrics = [{
        'jobname': '1_thread',
        'filesize': '50M',
        'iops': 95.26093,
        'bw': 97547,
        'lat_ns': {
            'min': 353377760,
            'max': 1697519869,
            'mean': 417754876.774692
        }
    }]

    extracted_metrics = self.fio_metrics_obj._extract_metrics(
        json_obj)

    self.assertEqual(expected_metrics, extracted_metrics)

  def test_extract_metrics_from_incomplete_files(self):
    """when input file contains a job with incomplete data.

    The partial_json file has non zero metric values for the 2nd job only.
    Since all metrics for 1st job have zero values, the 1st job will be ignored
    and only the 2nd job metrics will be returned
    """
    json_obj = self.fio_metrics_obj._load_file_dict(
        get_full_filepath(PARTIAL_FILE))
    expected_metrics = [{
        'jobname': '2_thread',
        'filesize': '50M',
        'iops': 95.26093,
        'bw': 97547,
        'lat_ns': {
            'min': 353377760,
            'max': 1697519869,
            'mean': 417754876.774692
        }
    }]

    extracted_metrics = self.fio_metrics_obj._extract_metrics(
        json_obj)

    self.assertEqual(expected_metrics, extracted_metrics)

  def test_extract_metrics_values_from_no_data_raises_no_values_error(self):
    """tests if extract_metrics() raises error if no metrics are extracted."""
    json_obj = self.fio_metrics_obj._load_file_dict(
        get_full_filepath(NO_METRICS_FILE))
    extracted_metrics = None

    with self.assertRaises(fio_metrics.NoValuesError):
      extracted_metrics = self.fio_metrics_obj._extract_metrics(
          json_obj)
    self.assertIsNone(extracted_metrics)

  def test_get_metrics_for_good_file(self):
    expected_metrics = [{
        'jobname': '1_thread',
        'filesize': '50M',
        'iops': 95.26093,
        'bw': 97547,
        'lat_ns': {
            'min': 353377760,
            'max': 1697519869,
            'mean': 417754876.774692
        }
    }]

    extracted_metrics = self.fio_metrics_obj.get_metrics(
        get_full_filepath(GOOD_FILE), False)

    self.assertEqual(expected_metrics, extracted_metrics)

  def test_get_metrics_for_multiple_jobs_global_fsize(self):
    """multiple_jobs_global_fsize_fpath has filesize as global parameter.

    """
    expected_metrics = [{
        'jobname': '1_thread',
        'filesize': '50M',
        'iops': 115.354741,
        'bw': 135655,
        'lat_ns': {
            'min': 249737264,
            'max': 28958587178,
            'mean': 18494668007.316742
        }
    }, {
        'jobname': '2_thread',
        'filesize': '50M',
        'iops': 34.641075,
        'bw': 40988,
        'lat_ns': {
            'min': 212007238,
            'max': 21590713209,
            'mean': 15969313013.822775
        }
    }]

    extracted_metrics = self.fio_metrics_obj.get_metrics(
        get_full_filepath(MULTIPLE_JOBS_GLOBAL_FSIZE_FILE), False)

    self.assertEqual(expected_metrics, extracted_metrics)

  def test_get_metrics_for_multiple_jobs_job_fsize(self):
    """multiple_jobs_global_fsize_fpath has filesize as job parameter.

    """
    expected_metrics = [{
        'jobname': '1_thread',
        'filesize': '3M',
        'iops': 88.851558,
        'bw': 103682,
        'lat_ns': {
            'min': 173373014,
            'max': 36442812445,
            'mean': 21799839057.909954
        }
    }, {
        'jobname': '2_thread',
        'filesize': '5M',
        'iops': 37.52206,
        'bw': 44249,
        'lat_ns': {
            'min': 172148734,
            'max': 20110704859,
            'mean': 14960429037.40382
        }
    }]

    extracted_metrics = self.fio_metrics_obj.get_metrics(
        get_full_filepath(MULTIPLE_JOBS_JOB_FSIZE_FILE), False)

    self.assertEqual(expected_metrics, extracted_metrics)


if __name__ == '__main__':
  googletest.main()
