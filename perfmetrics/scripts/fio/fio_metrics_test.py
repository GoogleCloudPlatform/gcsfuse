"""Tests for fio_metrics.

  Usage from perfmetrics/scripts folder: python3 -m fio.fio_metrics_test
"""
import unittest
from unittest import mock
from fio import fio_metrics

TEST_PATH = './fio/testdata/'
GOOD_FILE = 'good_out_job.json'
EMPTY_FILE = 'empty_file.json'
EMPTY_JSON_FILE = 'empty_json.json'
PARTIAL_FILE = 'partial_metrics.json'
NO_METRICS_FILE = 'no_metrics.json'
BAD_FORMAT_FILE = 'bad_format.json'
MULTIPLE_JOBS_GLOBAL_FSIZE_FILE = 'multiple_jobs_global_fsize.json'
MULTIPLE_JOBS_JOB_FSIZE_FILE = 'multiple_jobs_job_fsize.json'

SPREADSHEET_ID = '1kvHv1OBCzr9GnFxRu9RTJC7jjQjc9M4rAiDnhyak2Sg'
NUM_ENTRIES_CELL = 'T4'
WORKSHEET_NAME = 'fio_metrics!'


def get_full_filepath(filename):
  filepath = '{}{}'.format(TEST_PATH, filename)
  return filepath


class TestFioMetricsTest(unittest.TestCase):

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
    """Input file is not in JSON format.

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
        'filesize': 50000,
        'num_threads': 40,
        'rw': 'read',
        'start_time': 1653027084,
        'end_time': 1653027155,
        'iops': 95.26093,
        'bw_bytes': 99888324,
        'io_bytes': 6040846336,
        'lat_s': {
            'min': 0.35337776000000004,
            'max': 1.6975198690000002,
            'mean': 0.41775487677469203,
            'lat_20_perc': 0.37958451200000004,
            'lat_50_perc': 0.38797312,
            'lat_90_perc': 0.49283072000000006,
            'lat_95_perc': 0.526385152
        }
    }]

    extracted_metrics = self.fio_metrics_obj._extract_metrics(json_obj)

    self.assertEqual(expected_metrics, extracted_metrics)

  def test_extract_metrics_from_incomplete_files(self):
    """When input file contains a job with incomplete data.

    The partial_json file has non zero metric values for the 2nd job only.
    Since all metrics for 1st job have zero values, the 1st job will be ignored
    and only the 2nd job metrics will be returned
    """
    json_obj = self.fio_metrics_obj._load_file_dict(
        get_full_filepath(PARTIAL_FILE))
    expected_metrics = [{
        'jobname': '2_thread',
        'filesize': 50000,
        'num_threads': 40,
        'rw': 'read',
        'start_time': 1653027084,
        'end_time': 1653027155,
        'iops': 95.26093,
        'bw_bytes': 99888324,
        'io_bytes': 6040846336,
        'lat_s': {
            'min': 0.35337776000000004,
            'max': 1.6975198690000002,
            'mean': 0.41775487677469203,
            'lat_20_perc': 0.37958451200000004,
            'lat_50_perc': 0.38797312,
            'lat_90_perc': 0.49283072000000006,
            'lat_95_perc': 0.526385152
        }
    }]

    extracted_metrics = self.fio_metrics_obj._extract_metrics(json_obj)

    self.assertEqual(expected_metrics, extracted_metrics)

  def test_extract_metrics_values_from_no_data_raises_no_values_error(self):
    """Tests if extract_metrics() raises error if no metrics are extracted."""
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
        'filesize': 50000,
        'num_threads': 40,
        'rw': 'read',
        'start_time': 1653027084,
        'end_time': 1653027155,
        'iops': 95.26093,
        'bw_bytes': 99888324,
        'io_bytes': 6040846336,
        'lat_s': {
            'min': 0.35337776000000004,
            'max': 1.6975198690000002,
            'mean': 0.41775487677469203,
            'lat_20_perc': 0.37958451200000004,
            'lat_50_perc': 0.38797312,
            'lat_90_perc': 0.49283072000000006,
            'lat_95_perc': 0.526385152
        }
    }]
    get_response = {
        'range': '{}{}'.format(WORKSHEET_NAME, NUM_ENTRIES_CELL),
        'majorDimension': 'ROWS',
        'values': [['6']]
    }
    new_row = 8
    update_response = {
        'spreadsheetId': SPREADSHEET_ID,
        'updatedRange': '{0}A{1}:H{1}'.format(WORKSHEET_NAME, new_row),
        'updatedRows': 1,
        'updatedColumns': 15,
        'updatedCells': 15
    }
    sheets_service_mock = mock.MagicMock()
    sheets_service_mock.spreadsheets().values().get(
    ).execute.return_value = get_response
    sheets_service_mock.spreadsheets().values().update(
    ).execute.return_value = update_response
    calls = [
        mock.call.spreadsheets().values().get(
            spreadsheetId=SPREADSHEET_ID,
            range='{}{}'.format(WORKSHEET_NAME, NUM_ENTRIES_CELL)),
        mock.call.spreadsheets().values().update(
            spreadsheetId=SPREADSHEET_ID,
            valueInputOption='USER_ENTERED',
            body={
                'majorDimension':
                    'ROWS',
                'values': [('1_thread', 50000, 40, 'read', 1653027084, 1653027155,
                            95.26093, 99888324, 6040846336, 0.35337776000000004,
                            1.6975198690000002, 0.41775487677469203,
                            0.37958451200000004, 0.38797312,
                            0.49283072000000006, 0.526385152)]
            },
            range='{}A{}'.format(WORKSHEET_NAME, new_row))
    ]
    with mock.patch.object(fio_metrics.gsheet, '_get_sheets_service_client'
                           ) as get_sheets_service_client_mock:
      get_sheets_service_client_mock.return_value = sheets_service_mock
      extracted_metrics = self.fio_metrics_obj.get_metrics(
          get_full_filepath(GOOD_FILE))

    self.assertEqual(expected_metrics, extracted_metrics)
    sheets_service_mock.assert_has_calls(calls, any_order=True)

  def test_get_metrics_for_multiple_jobs_global_fsize(self):
    """Multiple_jobs_global_fsize_fpath has filesize as global parameter."""
    expected_metrics = [{
        'jobname': '1_thread',
        'filesize': 50000,
        'num_threads': 40,
        'rw': 'read',
        'start_time': 1653381687,
        'end_time': 1653381758,
        'iops': 115.354741,
        'bw_bytes': 138911322,
        'io_bytes': 8405385216,
        'lat_s': {
            'min': 0.24973726400000001,
            'max': 28.958587178000002,
            'mean': 18.494668007316744,
            'lat_20_perc': 0.37958451200000004,
            'lat_50_perc': 0.38797312,
            'lat_90_perc': 0.49283072000000006,
            'lat_95_perc': 0.526385152
        }
    }, {
        'jobname': '2_thread',
        'filesize': 50000,
        'num_threads': 10,
        'rw': 'read',
        'start_time': 1653381758,
        'end_time': 1653381828,
        'iops': 34.641075,
        'bw_bytes': 41972238,
        'io_bytes': 2532311040,
        'lat_s': {
            'min': 0.21200723800000001,
            'max': 21.590713209,
            'mean': 15.969313013822775,
            'lat_20_perc': 0.37958451200000004,
            'lat_50_perc': 0.38797312,
            'lat_90_perc': 0.49283072000000006,
            'lat_95_perc': 0.526385152
        }
    }]
    get_response = {
        'range': '{}{}'.format(WORKSHEET_NAME, NUM_ENTRIES_CELL),
        'majorDimension': 'ROWS',
        'values': [['6']]
    }
    new_row = 8
    update_response = {
        'spreadsheetId': SPREADSHEET_ID,
        'updatedRange': '{}A{}:H{}'.format(WORKSHEET_NAME, new_row, new_row+1),
        'updatedRows': 2,
        'updatedColumns': 15,
        'updatedCells': 30
    }
    sheets_service_mock = mock.MagicMock()
    sheets_service_mock.spreadsheets().values().get(
    ).execute.return_value = get_response
    sheets_service_mock.spreadsheets().values().update(
    ).execute.return_value = update_response
    calls = [
        mock.call.spreadsheets().values().get(
            spreadsheetId=SPREADSHEET_ID,
            range='{}{}'.format(WORKSHEET_NAME, NUM_ENTRIES_CELL)),
        mock.call.spreadsheets().values().update(
            spreadsheetId=SPREADSHEET_ID,
            valueInputOption='USER_ENTERED',
            body={
                'majorDimension':
                    'ROWS',
                'values': [
                    ('1_thread', 50000, 40, 'read', 1653381687, 1653381758, 115.354741,
                     138911322, 8405385216, 0.24973726400000001,
                     28.958587178000002, 18.494668007316744,
                     0.37958451200000004, 0.38797312, 0.49283072000000006,
                     0.526385152),
                    ('2_thread', 50000, 10, 'read', 1653381758, 1653381828, 34.641075,
                     41972238, 2532311040, 0.21200723800000001, 21.590713209,
                     15.969313013822775, 0.37958451200000004, 0.38797312,
                     0.49283072000000006, 0.526385152)
                ]
            },
            range='{}A{}'.format(WORKSHEET_NAME, new_row))
    ]
    with mock.patch.object(fio_metrics.gsheet, '_get_sheets_service_client'
                           ) as get_sheets_service_client_mock:
      get_sheets_service_client_mock.return_value = sheets_service_mock
      extracted_metrics = self.fio_metrics_obj.get_metrics(
          get_full_filepath(MULTIPLE_JOBS_GLOBAL_FSIZE_FILE))

    self.assertEqual(expected_metrics, extracted_metrics)
    sheets_service_mock.assert_has_calls(calls, any_order=True)

  def test_get_metrics_for_multiple_jobs_job_fsize(self):
    """Multiple_jobs_global_fsize_fpath has filesize as job parameter."""
    expected_metrics = [{
        'jobname': '1_thread',
        'filesize': 3000,
        'num_threads': 40,
        'rw': 'read',
        'start_time': 1653597009,
        'end_time': 1653597085,
        'iops': 88.851558,
        'bw_bytes': 106170722,
        'io_bytes': 6952058880,
        'lat_s': {
            'min': 0.17337301400000002,
            'max': 36.442812445,
            'mean': 21.799839057909956,
            'lat_20_perc': 0.37958451200000004,
            'lat_50_perc': 0.38797312,
            'lat_90_perc': 0.49283072000000006,
            'lat_95_perc': 0.526385152
        }
    }, {
        'jobname': '2_thread',
        'filesize': 5000,
        'num_threads': 10,
        'rw': 'read',
        'start_time': 1653597085,
        'end_time': 1653597156,
        'iops': 37.52206,
        'bw_bytes': 45311294,
        'io_bytes': 2747269120,
        'lat_s': {
            'min': 0.172148734,
            'max': 20.110704859000002,
            'mean': 14.960429037403822,
            'lat_20_perc': 0.37958451200000004,
            'lat_50_perc': 0.38797312,
            'lat_90_perc': 0.49283072000000006,
            'lat_95_perc': 0.526385152
        }
    }]

    get_response = {
        'range': '{}{}'.format(WORKSHEET_NAME, NUM_ENTRIES_CELL),
        'majorDimension': 'ROWS',
        'values': [['6']]
    }
    new_row = 8
    update_response = {
        'spreadsheetId': SPREADSHEET_ID,
        'updatedRange': '{}A{}:H{}'.format(WORKSHEET_NAME, new_row, new_row+1),
        'updatedRows': 2,
        'updatedColumns': 15,
        'updatedCells': 30
    }
    sheets_service_mock = mock.MagicMock()
    sheets_service_mock.spreadsheets().values().get(
    ).execute.return_value = get_response
    sheets_service_mock.spreadsheets().values().update(
    ).execute.return_value = update_response
    calls = [
        mock.call.spreadsheets().values().get(
            spreadsheetId=SPREADSHEET_ID,
            range='{}{}'.format(WORKSHEET_NAME, NUM_ENTRIES_CELL)),
        mock.call.spreadsheets().values().update(
            spreadsheetId=SPREADSHEET_ID,
            valueInputOption='USER_ENTERED',
            body={
                'majorDimension':
                    'ROWS',
                'values': [
                    ('1_thread', 3000, 40, 'read', 1653597009, 1653597085, 88.851558,
                     106170722, 6952058880, 0.17337301400000002, 36.442812445,
                     21.799839057909956, 0.37958451200000004, 0.38797312,
                     0.49283072000000006, 0.526385152),
                    ('2_thread', 5000, 10, 'read', 1653597085, 1653597156, 37.52206,
                     45311294, 2747269120, 0.172148734, 20.110704859000002,
                     14.960429037403822, 0.37958451200000004, 0.38797312,
                     0.49283072000000006, 0.526385152)
                ],
            },
            range='{}A{}'.format(WORKSHEET_NAME, new_row))
    ]

    with mock.patch.object(fio_metrics.gsheet, '_get_sheets_service_client'
                           ) as get_sheets_service_client_mock:
      get_sheets_service_client_mock.return_value = sheets_service_mock
      extracted_metrics = self.fio_metrics_obj.get_metrics(
          get_full_filepath(MULTIPLE_JOBS_JOB_FSIZE_FILE))

    self.assertEqual(expected_metrics, extracted_metrics)
    sheets_service_mock.assert_has_calls(calls, any_order=True)


if __name__ == '__main__':
  unittest.main()

