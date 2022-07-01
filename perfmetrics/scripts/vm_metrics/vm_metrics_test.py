"""Tests for vm_metrics."""
import json
import sys
from unittest import mock
import vm_metrics
from unittest import TestCase
import os

TEST_PATH = './vm_metrics/testdata'

CPU_UTI_METRIC = 'compute.googleapis.com/instance/cpu/utilization'
RECEIVED_BYTES_METRIC = 'compute.googleapis.com/instance/network/received_bytes_count'
OPS_ERROR_COUNT_METRIC = 'custom.googleapis.com/gcsfuse/fs/ops_error_count'
OPS_LATENCY_METRIC = 'custom.googleapis.com/gcsfuse/fs/ops_latency'
READ_BYTES_COUNT_METRIC = 'custom.googleapis.com/gcsfuse/gcs/read_bytes_count'


TEST_START_TIME_SEC = 1656300600
TEST_END_TIME_SEC = 1656301200
TEST_INSTANCE = 'drashti-load-test'
TEST_PERIOD = 120
CPU_UTI_FACTOR = 1 / 100
REC_BYTES_FACTOR = 60
OPS_ERROR_COUNT_FACTOR = 1
OPS_LATENCY_FACTOR = 1
READ_BYTES_COUNT_FACTOR = 1
PEAK_ALIGNER = 'ALIGN_MAX'
MEAN_ALIGNER = 'ALIGN_MEAN'
DELTA_ALIGNER = 'ALIGN_DELTA'
SUM_REDUCER = 'REDUCE_SUM'
GROUP_FIELDS = ['metric.labels']

REC_BYTES_MEAN_METRIC_POINT_1 = vm_metrics.MetricPoint(6566811.916666667, 1656300720, 1656300720)
REC_BYTES_MEAN_METRIC_POINT_2 = vm_metrics.MetricPoint(6772270.541666667, 1656300840, 1656300840)
REC_BYTES_MEAN_METRIC_POINT_3 = vm_metrics.MetricPoint(6918446.791666667, 1656300960, 1656300960)
REC_BYTES_MEAN_METRIC_POINT_4 = vm_metrics.MetricPoint(7114583.9, 1656301080, 1656301080)
REC_BYTES_MEAN_METRIC_POINT_5 = vm_metrics.MetricPoint(7138908.908333333, 1656301200, 1656301200)
EXPECTED_RECEIVED_BYTES_MEAN_DATA = [
    REC_BYTES_MEAN_METRIC_POINT_1, REC_BYTES_MEAN_METRIC_POINT_2,
    REC_BYTES_MEAN_METRIC_POINT_3, REC_BYTES_MEAN_METRIC_POINT_4, REC_BYTES_MEAN_METRIC_POINT_5
]

REC_BYTES_PEAK_METRIC_POINT_1 = vm_metrics.MetricPoint(6685105.283333333, 1656300720, 1656300720)
REC_BYTES_PEAK_METRIC_POINT_2 = vm_metrics.MetricPoint(6803372.233333333, 1656300840, 1656300840)
REC_BYTES_PEAK_METRIC_POINT_3 = vm_metrics.MetricPoint(6933473.3, 1656300960, 1656300960)
REC_BYTES_PEAK_METRIC_POINT_4 = vm_metrics.MetricPoint(7282851.5, 1656301080, 1656301080)
REC_BYTES_PEAK_METRIC_POINT_5 = vm_metrics.MetricPoint(7395036.616666666, 1656301200, 1656301200)
EXPECTED_RECEIVED_BYTES_PEAK_DATA = [
    REC_BYTES_PEAK_METRIC_POINT_1, REC_BYTES_PEAK_METRIC_POINT_2,
    REC_BYTES_PEAK_METRIC_POINT_3, REC_BYTES_PEAK_METRIC_POINT_4, REC_BYTES_PEAK_METRIC_POINT_5
]

CPU_UTI_MEAN_METRIC_POINT_1 = vm_metrics.MetricPoint(22.022823358129244, 1656300720, 1656300720)
CPU_UTI_MEAN_METRIC_POINT_2 = vm_metrics.MetricPoint(23.330100279029768, 1656300840, 1656300840)
CPU_UTI_MEAN_METRIC_POINT_3 = vm_metrics.MetricPoint(23.58245408118819, 1656300960, 1656300960)
CPU_UTI_MEAN_METRIC_POINT_4 = vm_metrics.MetricPoint(24.508291550769677, 1656301080, 1656301080)
CPU_UTI_MEAN_METRIC_POINT_5 = vm_metrics.MetricPoint(23.383308206167484, 1656301200, 1656301200)
EXPECTED_CPU_UTI_MEAN_DATA = [
    CPU_UTI_MEAN_METRIC_POINT_1, CPU_UTI_MEAN_METRIC_POINT_2, CPU_UTI_MEAN_METRIC_POINT_3,
    CPU_UTI_MEAN_METRIC_POINT_4, CPU_UTI_MEAN_METRIC_POINT_5
]

CPU_UTI_PEAK_METRIC_POINT_1 = vm_metrics.MetricPoint(22.053231452171328, 1656300720, 1656300720)
CPU_UTI_PEAK_METRIC_POINT_2 = vm_metrics.MetricPoint(23.417254448480286, 1656300840, 1656300840)
CPU_UTI_PEAK_METRIC_POINT_3 = vm_metrics.MetricPoint(23.810199799611127, 1656300960, 1656300960)
CPU_UTI_PEAK_METRIC_POINT_4 = vm_metrics.MetricPoint(25.30334780231998, 1656301080, 1656301080)
CPU_UTI_PEAK_METRIC_POINT_5 = vm_metrics.MetricPoint(24.29497897243386, 1656301200, 1656301200)
EXPECTED_CPU_UTI_PEAK_DATA = [
    CPU_UTI_PEAK_METRIC_POINT_1, CPU_UTI_PEAK_METRIC_POINT_2, CPU_UTI_PEAK_METRIC_POINT_3,
    CPU_UTI_PEAK_METRIC_POINT_4, CPU_UTI_PEAK_METRIC_POINT_5
]

OPS_ERROR_COUNT_METRIC_POINT_1 = vm_metrics.MetricPoint(95.0, 1656300600, 1656300720)
OPS_ERROR_COUNT_METRIC_POINT_2 = vm_metrics.MetricPoint(235.0, 1656300720, 1656300840)
OPS_ERROR_COUNT_METRIC_POINT_3 = vm_metrics.MetricPoint(100.0, 1656300840, 1656300960)
OPS_ERROR_COUNT_METRIC_POINT_4 = vm_metrics.MetricPoint(157.0, 1656300960, 1656301080)
OPS_ERROR_COUNT_METRIC_POINT_5 = vm_metrics.MetricPoint(165.0, 1656301080, 1656301200)
EXPECTED_OPS_ERROR_COUNT_DATA = [
    OPS_ERROR_COUNT_METRIC_POINT_1, OPS_ERROR_COUNT_METRIC_POINT_2, OPS_ERROR_COUNT_METRIC_POINT_3,
    OPS_ERROR_COUNT_METRIC_POINT_4, OPS_ERROR_COUNT_METRIC_POINT_5
]

READ_BYTES_COUNT_METRIC_POINT_1 = vm_metrics.MetricPoint(725685157.0, 1656300600, 1656300720)
READ_BYTES_COUNT_METRIC_POINT_2 = vm_metrics.MetricPoint(746803219.0, 1656300720, 1656300840)
READ_BYTES_COUNT_METRIC_POINT_3 = vm_metrics.MetricPoint(759282126.0, 1656300840, 1656300960)
READ_BYTES_COUNT_METRIC_POINT_4 = vm_metrics.MetricPoint(788001964.0, 1656300960, 1656301080)
READ_BYTES_COUNT_METRIC_POINT_5 = vm_metrics.MetricPoint(781867592.0, 1656301080, 1656301200)
EXPECTED_READ_BYTES_COUNT_DATA = [
    READ_BYTES_COUNT_METRIC_POINT_1, READ_BYTES_COUNT_METRIC_POINT_2, READ_BYTES_COUNT_METRIC_POINT_3,
    READ_BYTES_COUNT_METRIC_POINT_4, READ_BYTES_COUNT_METRIC_POINT_5
]

OPS_LATENCY_MEAN_METRIC_POINT_1 = vm_metrics.MetricPoint(15.40791568806023, 1656300600, 1656300720)
OPS_LATENCY_MEAN_METRIC_POINT_2 = vm_metrics.MetricPoint(14.968170482459712, 1656300720, 1656300840)
OPS_LATENCY_MEAN_METRIC_POINT_3 = vm_metrics.MetricPoint(15.080919708390327, 1656300840, 1656300960)
OPS_LATENCY_MEAN_METRIC_POINT_4 = vm_metrics.MetricPoint(14.724381767456052, 1656300960, 1656301080)
OPS_LATENCY_MEAN_METRIC_POINT_5 = vm_metrics.MetricPoint(14.73861060219869, 1656301080, 1656301200)
EXPECTED_OPS_LATENCY_MEAN_DATA = [
    OPS_LATENCY_MEAN_METRIC_POINT_1, OPS_LATENCY_MEAN_METRIC_POINT_2, OPS_LATENCY_MEAN_METRIC_POINT_3,
    OPS_LATENCY_MEAN_METRIC_POINT_4, OPS_LATENCY_MEAN_METRIC_POINT_5
]

EXPECTED_ZERO_DATA = [vm_metrics.MetricPoint(0, 0, 0) for i in range(int((TEST_END_TIME_SEC-TEST_START_TIME_SEC)/TEST_PERIOD)+1)]

class MetricsResponseObject:

  def __init__(self, dict1):
    self.__dict__.update(dict1)


def dict_to_obj(dict1):
  """Converting dictionary into object."""
  return json.loads(json.dumps(dict1), object_hook=MetricsResponseObject)


def get_response_from_filename(filename):
  response_filepath = os.path.join(TEST_PATH, filename + '.json')
  metrics_file = open(response_filepath, 'r')
  metrics_response = dict_to_obj(json.load(metrics_file))
  return metrics_response


class VmmetricsTest(TestCase):

  def setUp(self):
    super().setUp()
    self.vm_metrics_obj = vm_metrics.VmMetrics()

  def test_validate_start_end_times_with_start_time_greater_than_end_time(self):
    with self.assertRaises(ValueError):
      self.vm_metrics_obj._validate_start_end_times('1600000', '150000')

  def test_validate_start_end_times_throws_no_error(self):
    self.assertEqual(
        True,
        self.vm_metrics_obj._validate_start_end_times('1600000', '1600300'))

  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_cpu_utilization_peak_throws_no_values_error(
      self, mock_get_api_response):
    mock_get_api_response.return_value = {}

    with self.assertRaises(vm_metrics.NoValuesError):
      self.vm_metrics_obj._get_metrics(TEST_START_TIME_SEC, TEST_END_TIME_SEC,
                                       TEST_INSTANCE, TEST_PERIOD,
                                       CPU_UTI_METRIC, CPU_UTI_FACTOR, PEAK_ALIGNER)
  
  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_cpu_utilization_mean_throws_no_values_error(
      self, mock_get_api_response):
    mock_get_api_response.return_value = {}

    with self.assertRaises(vm_metrics.NoValuesError):
      self.vm_metrics_obj._get_metrics(TEST_START_TIME_SEC, TEST_END_TIME_SEC,
                                       TEST_INSTANCE, TEST_PERIOD,
                                       CPU_UTI_METRIC, CPU_UTI_FACTOR, MEAN_ALIGNER)

  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_received_bytes_peak_throws_no_values_error(
      self, mock_get_api_response):
    mock_get_api_response.return_value = {}

    with self.assertRaises(vm_metrics.NoValuesError):
      self.vm_metrics_obj._get_metrics(TEST_START_TIME_SEC, TEST_END_TIME_SEC,
                                       TEST_INSTANCE, TEST_PERIOD,
                                       RECEIVED_BYTES_METRIC, REC_BYTES_FACTOR, PEAK_ALIGNER)
  
  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_received_bytes_mean_throws_no_values_error(
      self, mock_get_api_response):
    mock_get_api_response.return_value = {}

    with self.assertRaises(vm_metrics.NoValuesError):
      self.vm_metrics_obj._get_metrics(TEST_START_TIME_SEC, TEST_END_TIME_SEC,
                                       TEST_INSTANCE, TEST_PERIOD,
                                       RECEIVED_BYTES_METRIC, REC_BYTES_FACTOR, MEAN_ALIGNER)

  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_ops_latency_mean_throws_no_values_error(
      self, mock_get_api_response):
    mock_get_api_response.return_value = {}

    with self.assertRaises(vm_metrics.NoValuesError):
      self.vm_metrics_obj._get_metrics(TEST_START_TIME_SEC, TEST_END_TIME_SEC,
                                       TEST_INSTANCE, TEST_PERIOD,
                                       OPS_LATENCY_METRIC, OPS_LATENCY_FACTOR, DELTA_ALIGNER)
  
  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_ops_read_bytes_count_throws_no_values_error(
      self, mock_get_api_response):
    mock_get_api_response.return_value = {}

    with self.assertRaises(vm_metrics.NoValuesError):
      self.vm_metrics_obj._get_metrics(TEST_START_TIME_SEC, TEST_END_TIME_SEC,
                                       TEST_INSTANCE, TEST_PERIOD,
                                       READ_BYTES_COUNT_METRIC, READ_BYTES_COUNT_FACTOR, DELTA_ALIGNER)
  
  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_ops_error_count_returns_list_of_zeroes(
      self, mock_get_api_response):
    mock_get_api_response.return_value = {}

    ops_error_count_data = self.vm_metrics_obj._get_metrics(
        TEST_START_TIME_SEC, TEST_END_TIME_SEC, TEST_INSTANCE, TEST_PERIOD,
        OPS_ERROR_COUNT_METRIC, OPS_ERROR_COUNT_FACTOR, DELTA_ALIGNER, SUM_REDUCER, GROUP_FIELDS)

    self.assertEqual(ops_error_count_data, EXPECTED_ZERO_DATA)

  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_cpu_utilization_peak(self, mock_get_api_response):
    metrics_response = get_response_from_filename(
        'peak_cpu_utilization_response')
    mock_get_api_response.return_value = [metrics_response]

    cpu_uti_peak_data = self.vm_metrics_obj._get_metrics(TEST_START_TIME_SEC,
                                                    TEST_END_TIME_SEC,
                                                    TEST_INSTANCE, TEST_PERIOD,
                                                    CPU_UTI_METRIC,
                                                    CPU_UTI_FACTOR, PEAK_ALIGNER)

    self.assertEqual(cpu_uti_peak_data, EXPECTED_CPU_UTI_PEAK_DATA)

  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_cpu_utilization_mean(self, mock_get_api_response):
    metrics_response = get_response_from_filename(
        'mean_cpu_utilization_response')
    mock_get_api_response.return_value = [metrics_response]

    cpu_uti_mean_data = self.vm_metrics_obj._get_metrics(TEST_START_TIME_SEC,
                                                    TEST_END_TIME_SEC,
                                                    TEST_INSTANCE, TEST_PERIOD,
                                                    CPU_UTI_METRIC,
                                                    CPU_UTI_FACTOR, MEAN_ALIGNER)

    self.assertEqual(cpu_uti_mean_data, EXPECTED_CPU_UTI_MEAN_DATA)

  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_received_bytes_peak(self, mock_get_api_response):
    metrics_response = get_response_from_filename(
        'peak_received_bytes_count_response')
    mock_get_api_response.return_value = [metrics_response]

    rec_bytes_peak_data = self.vm_metrics_obj._get_metrics(
        TEST_START_TIME_SEC, TEST_END_TIME_SEC, TEST_INSTANCE, TEST_PERIOD,
        RECEIVED_BYTES_METRIC, REC_BYTES_FACTOR, PEAK_ALIGNER)

    self.assertEqual(rec_bytes_peak_data, EXPECTED_RECEIVED_BYTES_PEAK_DATA)

  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_received_bytes_mean(self, mock_get_api_response):
    metrics_response = get_response_from_filename(
        'mean_received_bytes_count_response')
    mock_get_api_response.return_value = [metrics_response]

    rec_bytes_mean_data = self.vm_metrics_obj._get_metrics(
        TEST_START_TIME_SEC, TEST_END_TIME_SEC, TEST_INSTANCE, TEST_PERIOD,
        RECEIVED_BYTES_METRIC, REC_BYTES_FACTOR, MEAN_ALIGNER)

    self.assertEqual(rec_bytes_mean_data, EXPECTED_RECEIVED_BYTES_MEAN_DATA)
  

  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_ops_mean_latency(self, mock_get_api_response):
    metrics_response = get_response_from_filename(
        'ops_mean_latency_response')
    mock_get_api_response.return_value = [metrics_response]

    ops_latency_mean_data = self.vm_metrics_obj._get_metrics(
        TEST_START_TIME_SEC, TEST_END_TIME_SEC, TEST_INSTANCE, TEST_PERIOD,
        OPS_LATENCY_METRIC, OPS_LATENCY_FACTOR, DELTA_ALIGNER)

    self.assertEqual(ops_latency_mean_data, EXPECTED_OPS_LATENCY_MEAN_DATA)
  
  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_read_bytes_count(self, mock_get_api_response):
    metrics_response = get_response_from_filename(
        'read_bytes_count_response')
    mock_get_api_response.return_value = [metrics_response]

    read_bytes_count_data = self.vm_metrics_obj._get_metrics(
        TEST_START_TIME_SEC, TEST_END_TIME_SEC, TEST_INSTANCE, TEST_PERIOD,
        READ_BYTES_COUNT_METRIC, READ_BYTES_COUNT_FACTOR, DELTA_ALIGNER)

    self.assertEqual(read_bytes_count_data, EXPECTED_READ_BYTES_COUNT_DATA)
  
  @mock.patch.object(vm_metrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_ops_error_count(self, mock_get_api_response):
    metrics_response = get_response_from_filename(
        'ops_error_count_response')
    mock_get_api_response.return_value = [metrics_response]

    ops_error_count_data = self.vm_metrics_obj._get_metrics(
        TEST_START_TIME_SEC, TEST_END_TIME_SEC, TEST_INSTANCE, TEST_PERIOD,
        OPS_ERROR_COUNT_METRIC, OPS_ERROR_COUNT_FACTOR, DELTA_ALIGNER, SUM_REDUCER, GROUP_FIELDS)

    self.assertEqual(ops_error_count_data, EXPECTED_OPS_ERROR_COUNT_DATA)
