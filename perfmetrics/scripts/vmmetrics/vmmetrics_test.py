"""Tests for vmmetrics."""
import json
import sys
from unittest import mock
import vmmetrics
from unittest import TestCase
import os

TEST_PATH = './vmmetrics/testdata'

CPU_UTI_METRIC = 'compute.googleapis.com/instance/cpu/utilization'
RECEIVED_BYTES_METRIC = 'compute.googleapis.com/instance/network/received_bytes_count'

REC_BYTES_METRIC_POINT_1 = vmmetrics.MetricPoint(15.2182, 15.097991666666667,
                                                 1652699400, 1652699400)
REC_BYTES_METRIC_POINT_2 = vmmetrics.MetricPoint(15.951266666666667, 15.43835,
                                                 1652699280, 1652699280)
REC_BYTES_METRIC_POINT_3 = vmmetrics.MetricPoint(14.235966666666666,
                                                 13.995783333333334, 1652699160,
                                                 1652699160)
REC_BYTES_METRIC_POINT_4 = vmmetrics.MetricPoint(15.26295, 14.97935, 1652699040,
                                                 1652699040)
REC_BYTES_METRIC_POINT_5 = vmmetrics.MetricPoint(15.324016666666667, 14.736275,
                                                 1652698920, 1652698920)
EXPECTED_RECEIVED_BYTES_DATA = [
    REC_BYTES_METRIC_POINT_5, REC_BYTES_METRIC_POINT_4,
    REC_BYTES_METRIC_POINT_3, REC_BYTES_METRIC_POINT_2, REC_BYTES_METRIC_POINT_1
]

CPU_UTI_METRIC_POINT_1 = vmmetrics.MetricPoint(8.399437243594244,
                                               8.351115843785617, 1652699400,
                                               1652699400)
CPU_UTI_METRIC_POINT_2 = vmmetrics.MetricPoint(8.309971587877953,
                                               8.258347968997745, 1652699280,
                                               1652699280)
CPU_UTI_METRIC_POINT_3 = vmmetrics.MetricPoint(8.459197695240922,
                                               8.333482699502687, 1652699160,
                                               1652699160)
CPU_UTI_METRIC_POINT_4 = vmmetrics.MetricPoint(8.511909491062397,
                                               8.459746438844984, 1652699040,
                                               1652699040)
CPU_UTI_METRIC_POINT_5 = vmmetrics.MetricPoint(8.437140491150785,
                                               8.351230117089775, 1652698920,
                                               1652698920)
EXPECTED_CPU_UTI_DATA = [
    CPU_UTI_METRIC_POINT_5, CPU_UTI_METRIC_POINT_4, CPU_UTI_METRIC_POINT_3,
    CPU_UTI_METRIC_POINT_2, CPU_UTI_METRIC_POINT_1
]

TEST_START_TIME_SEC = 1652698800
TEST_END_TIME_SEC = 1652699400
TEST_INSTANCE = 'gke-cluster-1-default-pool-b89264dc-2633'
TEST_PERIOD = 120
CPU_UTI_FACTOR = 1 / 100
REC_BYTES_FACTOR = 60000

METRICS_DATA = [[
    1652699400, 8.399437243594244, 8.351115843785617, 15.2182,
    15.097991666666667
],
                [
                    1652699280, 8.309971587877953, 8.258347968997745,
                    15.951266666666667, 15.43835
                ],
                [
                    1652699160, 8.459197695240922, 8.333482699502687,
                    14.235966666666666, 13.995783333333334
                ],
                [
                    1652699040, 8.511909491062397, 8.459746438844984, 15.26295,
                    14.97935
                ],
                [
                    1652698920, 8.437140491150785, 8.351230117089775,
                    15.324016666666667, 14.736275
                ]]


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
    self.vm_metrics_obj = vmmetrics.VmMetrics()

  def test_validate_start_end_times_with_start_time_greater_than_end_time(self):
    with self.assertRaises(ValueError):
      self.vm_metrics_obj._validate_start_end_times('1600000', '150000')

  def test_validate_start_end_times_throws_no_error(self):
    self.assertEqual(
        True,
        self.vm_metrics_obj._validate_start_end_times('1600000', '1600300'))

  @mock.patch.object(vmmetrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_cpu_utilization_throws_no_values_error(
      self, mock_get_api_response):
    mock_get_api_response.return_value = [{}, {}]

    with self.assertRaises(vmmetrics.NoValuesError):
      self.vm_metrics_obj._get_metrics(TEST_START_TIME_SEC, TEST_END_TIME_SEC,
                                       TEST_INSTANCE, TEST_PERIOD,
                                       CPU_UTI_METRIC, CPU_UTI_FACTOR)

  @mock.patch.object(vmmetrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_received_bytes_throws_no_values_error(
      self, mock_get_api_response):
    mock_get_api_response.return_value = [{}, {}]

    with self.assertRaises(vmmetrics.NoValuesError):
      self.vm_metrics_obj._get_metrics(TEST_START_TIME_SEC, TEST_END_TIME_SEC,
                                       TEST_INSTANCE, TEST_PERIOD,
                                       RECEIVED_BYTES_METRIC, REC_BYTES_FACTOR)

  @mock.patch.object(vmmetrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_cpu_utilization(self, mock_get_api_response):
    peak_metrics_response = get_response_from_filename(
        'peak_cpu_utilization_response')
    mean_metrics_response = get_response_from_filename(
        'mean_cpu_utilization_response')
    mock_get_api_response.return_value = [[peak_metrics_response],
                                          [mean_metrics_response]]

    cpu_uti_data = self.vm_metrics_obj._get_metrics(TEST_START_TIME_SEC,
                                                    TEST_END_TIME_SEC,
                                                    TEST_INSTANCE, TEST_PERIOD,
                                                    CPU_UTI_METRIC,
                                                    CPU_UTI_FACTOR)

    self.assertEqual(cpu_uti_data, EXPECTED_CPU_UTI_DATA)

  @mock.patch.object(vmmetrics.VmMetrics, '_get_api_response')
  def test_get_metrics_for_received_bytes(self, mock_get_api_response):
    peak_metrics_response = get_response_from_filename(
        'peak_received_bytes_count_response')
    mean_metrics_response = get_response_from_filename(
        'mean_received_bytes_count_response')
    mock_get_api_response.return_value = [[peak_metrics_response],
                                          [mean_metrics_response]]

    rec_bytes_data = self.vm_metrics_obj._get_metrics(
        TEST_START_TIME_SEC, TEST_END_TIME_SEC, TEST_INSTANCE, TEST_PERIOD,
        RECEIVED_BYTES_METRIC, REC_BYTES_FACTOR)

    self.assertEqual(rec_bytes_data, EXPECTED_RECEIVED_BYTES_DATA)
