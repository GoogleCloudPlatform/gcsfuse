"""Extracts required Google Cloud VM metrics using Monitoring V3 API call, parses
   the API response into a list and writes to google sheet.

   Takes VM instance name, interval start time, interval end time, alignment
   period, fio test type and google worksheet name as command line inputs.
   The supported fio test types are: read, write
   Metrics extracted:
   1.Peak Cpu Utilization(%)
   2.Mean Cpu Utilization(%)
   3.Peak Network Bandwidth(Bytes/s)
   4.Mean Network Bandwidth(Bytes/s)
   5.Read Bytes Count(Bytes)
   6.Opencensus Error Count
   7.Opencensus Mean Latency(s)

  Usage:
  >>python3 vm_metrics.py {instance} {start time in epoch sec} {end time in epoch sec} {period in sec} {test_type} {worksheet_name}

"""
import dataclasses
from dataclasses import dataclass, field
import os
import sys
import google.api_core
from google.api_core.exceptions import GoogleAPICallError
import google.cloud
from google.cloud import monitoring_v3
from gsheet import gsheet
from typing import List

PROJECT_NAME = 'projects/gcs-fuse-test-ml'
CPU_UTI_METRIC_TYPE = 'compute.googleapis.com/instance/cpu/utilization'
RECEIVED_BYTES_COUNT_METRIC_TYPE = 'compute.googleapis.com/instance/network/received_bytes_count'
OPS_LATENCY_METRIC_TYPE = 'custom.googleapis.com/gcsfuse/fs/ops_latency'
READ_BYTES_COUNT_METRIC_TYPE = 'custom.googleapis.com/gcsfuse/gcs/read_bytes_count'
OPS_ERROR_COUNT_METRIC_TYPE = 'custom.googleapis.com/gcsfuse/fs/ops_error_count'

@dataclasses.dataclass
class MetricPoint:
  value: float
  start_time_sec: int
  end_time_sec: int

'''Refer this doc to find appropriate values for the attributes of the Metric class: 
https://cloud.google.com/monitoring/custom-metrics/reading-metrics .'''
@dataclasses.dataclass
class Metric:
  metric_type: str
  factor: float
  aligner: str
  extra_filter: str = ''
  reducer: str = 'REDUCE_NONE'
  group_fields: List[str] = field(default_factory=list)
  metric_point_list: List[MetricPoint] = field(default_factory=list)


CPU_UTI_PEAK = Metric(
    metric_type=CPU_UTI_METRIC_TYPE, factor=1 / 100, aligner='ALIGN_MAX')
CPU_UTI_MEAN = Metric(
    metric_type=CPU_UTI_METRIC_TYPE, factor=1 / 100, aligner='ALIGN_MEAN')
REC_BYTES_PEAK = Metric(
    metric_type=RECEIVED_BYTES_COUNT_METRIC_TYPE,
    factor=60,
    aligner='ALIGN_MAX')
REC_BYTES_MEAN = Metric(
    metric_type=RECEIVED_BYTES_COUNT_METRIC_TYPE,
    factor=60,
    aligner='ALIGN_MEAN')
READ_BYTES_COUNT = Metric(
    metric_type=READ_BYTES_COUNT_METRIC_TYPE, factor=1, aligner='ALIGN_DELTA')

OPS_ERROR_COUNT_FILTER = 'metric.labels.fs_op != "GetXattr"'
OPS_ERROR_COUNT = Metric(
    metric_type=OPS_ERROR_COUNT_METRIC_TYPE,
    factor=1,
    aligner='ALIGN_DELTA',
    extra_filter=OPS_ERROR_COUNT_FILTER,
    reducer='REDUCE_SUM',
    group_fields=['metric.labels'])

METRICS_LIST = [
    CPU_UTI_PEAK, CPU_UTI_MEAN, REC_BYTES_PEAK, REC_BYTES_MEAN,
    READ_BYTES_COUNT, OPS_ERROR_COUNT
]


class NoValuesError(Exception):
  """API response values are missing."""

def _parse_metric_value_by_type(value, value_type) -> float:
  """Parses the value from a value object in API response.

    Args:
      value (object): The value object from API response
      value_type (int) : Integer representing the value type of the object, refer 
                        https://cloud.google.com/monitoring/api/ref_v3/rest/v3/TypedValue.
  """
  if value_type == 1:
    return value.bool_value
  elif value_type == 2:
    return value.int64_value
  elif value_type == 3:
    return value.double_value
  elif value_type == 4:
    return value.string_value
  elif value_type == 5:
    return value.distribution_value.mean
  else:
    raise Exception('Unhandled Value type')

def _get_metric_filter(type, metric_type, instance, extra_filter):
  """Getting the metrics filter string from metric type, instance name and extra filter.

    Args:
      metric_type (str): The type of metric, Metric.metric_type
      instance (str): VM instance name
      extra_filter(str): Metric.extra_filter
  """
  if (type == 'compute'):
    metric_filter = (
        'metric.type = "{metric_type}" AND metric.label.instance_name '
        '={instance_name}').format(
            metric_type=metric_type, instance_name=instance)
  elif (type == 'custom'):
    metric_filter = (
        'metric.type = "{metric_type}" AND metric.labels.opencensus_task = '
        'ends_with("{instance_name}")').format(
            metric_type=metric_type, instance_name=instance)

  if (extra_filter == ''):
    return metric_filter
  return '{} AND {}'.format(metric_filter, extra_filter)

def _create_metric_points_from_response(metrics_response, factor):
  """Parses the given metrics API response and returns a list of MetricPoint.

    Args:
      metrics_response (object): The metrics API response
      factor (float) : Converting the API response values into appropriate unit
    Returns:
      list[MetricPoint]
  """
  metric_point_list = []
  for metric in metrics_response:
    for point in metric.points:
      value = _parse_metric_value_by_type(point.value, metric.value_type)
      metric_point = MetricPoint(value / factor,
                                 point.interval.start_time.timestamp(),
                                 point.interval.end_time.timestamp())

      metric_point_list.append(metric_point)
  metric_point_list.reverse()
  return metric_point_list


class VmMetrics:

  def _validate_start_end_times(self, start_time_sec, end_time_sec):
    """Checks whether the start time is less than end time.

    Args:
      start_time_sec (int) : Epoch seconds
      end_time_sec (int) : Epoch seconds
    Raises:
      ValueError : When start time is after end time.
    """
    if start_time_sec < end_time_sec:
      return True
    else:
      raise ValueError('Start time should be before end time')

  def _get_api_response(self, start_time_sec, end_time_sec, instance, period, metric):
    """Fetches the API response for the requested metrics.

    Args:
      start_time_sec (int): Epoch seconds
      end_time_sec (int): Epoch seconds
      instance (str): VM instance name
      period (float): Period over which the values are aligned
      metric: Metric object
    Returns:
      metrics API response (object)
    Raises:
      GoogleAPICallError

    """

    client = monitoring_v3.MetricServiceClient()
    interval = monitoring_v3.TimeInterval(
        end_time={'seconds': int(end_time_sec)},
        start_time={'seconds': int(start_time_sec)})

    aggregation = monitoring_v3.Aggregation(
        alignment_period={'seconds': period},
        per_series_aligner=getattr(monitoring_v3.Aggregation.Aligner, metric.aligner),
        cross_series_reducer=getattr(monitoring_v3.Aggregation.Reducer,metric.reducer),
        group_by_fields=metric.group_fields
    )

    # Checking whether the metric is custom or compute by getting the first 6 or 7 elements of metric type:
    if (metric.metric_type[0:7] == 'compute'):
      metric_filter = _get_metric_filter('compute', metric.metric_type,
                                         instance, metric.extra_filter)
    elif (metric.metric_type[0:6] == 'custom'):
      metric_filter = _get_metric_filter('custom', metric.metric_type, instance,
                                         metric.extra_filter)
    else:
      raise Exception('Unhandled metric type')

    try:
      metrics_response = client.list_time_series({
          'name': PROJECT_NAME,
          'filter': metric_filter,
          'interval': interval,
          'view': monitoring_v3.ListTimeSeriesRequest.TimeSeriesView.FULL,
          'aggregation': aggregation,
      })
      
    except:
      raise GoogleAPICallError(('The request for API response of {} failed.'
                                ).format(metric.metric_type))

    return metrics_response

  def _get_metrics(self, start_time_sec, end_time_sec, instance, period,
                   metric):
    """Returns the MetricPoint list for requested metric type.

    Args:
      start_time_sec (int): Epoch seconds
      end_time_sec (int): Epoch seconds
      instance (str): VM instance name
      period (float): Period over which the values are aligned
      metric: Metric Object
    Returns:
      list[MetricPoint]
    """
    metrics_response = self._get_api_response(start_time_sec, end_time_sec,
                                              instance, period, metric)
    metrics_data = _create_metric_points_from_response(metrics_response,
                                                       metric.factor)

    # In case OPS_ERROR_COUNT data is empty, we return a list of zeroes:
    if(metric == OPS_ERROR_COUNT and metrics_data == []):
      return [MetricPoint(0, 0, 0) for i in range(int((end_time_sec-start_time_sec)/period)+1)]

    # Metrics data for metrics other that OPS_ERROR_COUNT_DATA should not be empty:
    if (metrics_data == []):
      raise NoValuesError('No values were retrieved from the call for ' +
                          metric.metric_type)

    return metrics_data
  
  def _add_new_metric_using_test_type(self, test_type):
    """Creates a copy of METRICS_LIST and appends new Metric objects to it.

    Args:
      test_type(str): The type of load test for which metrics are taken
    Returns:
      list[Metric]
    """
    # Getting the fs_op type from test_type:
    if test_type == 'read' or test_type == 'randread':
        fs_op = 'ReadFile'
    elif test_type == 'write' or test_type == 'randwrite':
        fs_op = 'WriteFile'

    updated_metrics_list = list(METRICS_LIST)

    # Creating a new metric that requires test_type and adding it to the updated_metrics_list:
    ops_latency_filter = 'metric.labels.fs_op = "{}"'.format(fs_op)
    ops_latency_mean = Metric(
        metric_type=OPS_LATENCY_METRIC_TYPE,
        extra_filter=ops_latency_filter,
        factor=1,
        aligner='ALIGN_DELTA')

    updated_metrics_list.append(ops_latency_mean)

    return updated_metrics_list

  def fetch_metrics(self, start_time_sec, end_time_sec, instance, period, test_type):
    """Fetches the metrics data for all types and returns a list of lists to be written in google sheet.

    Args:
      start_time_sec (int): Epoch seconds
      end_time_sec (int): Epoch seconds
      instance (str): VM instance
      period (float): Period over which the values are taken 
      test_type(str): The type of load test for which metrics are taken

    Returns:
      list[[period end time, interval end time, CPU_UTI_PEAK, CPU_UTI_MEAN,
      REC_BYTES_PEAK, REC_BYTES_MEAN, READ_BYTES_COUNT, OPS_ERROR_COUNT,
      OPS_MEAN_LATENCY]]
    """
    self._validate_start_end_times(start_time_sec, end_time_sec)
    
    # Getting updated metrics list:
    updated_metrics_list = self._add_new_metric_using_test_type(test_type)

    # Extracting MetricPoint list for every metric in the updated_metrics_list:
    for metric in updated_metrics_list:
      metric.metric_point_list = self._get_metrics(start_time_sec, end_time_sec,
                                                   instance, period, metric)

    # Creating a list of lists to write into google sheet:
    num_points = len(updated_metrics_list[0].metric_point_list)
    metrics_data = []
    for i in range(num_points):
      row = [updated_metrics_list[0].metric_point_list[i].start_time_sec]
      row.append(end_time_sec)
      for metric in updated_metrics_list:
        row.append(metric.metric_point_list[i].value)
      metrics_data.append(row)

    return metrics_data

  def fetch_metrics_and_write_to_google_sheet(self, start_time_sec,
                                              end_time_sec, instance, period,
                                              test_type, worksheet_name):
    """Fetches the metrics data for all types and writes to a google sheet.

    Args:
      start_time_sec (int): Epoch seconds
      end_time_sec (int): Epoch seconds
      instance (str): VM instance
      period (float): Period over which the values are taken
      test_type(str): The type of load test for which metrics are taken
      worksheet_name(str): The name of the google worksheet you want to write to
    Returns:
      None
    """
    self._validate_start_end_times(start_time_sec, end_time_sec)
    
    # Getting metrics data:
    metrics_data = self.fetch_metrics(start_time_sec, end_time_sec, instance,
                                      period, test_type)

    # Writing data into google sheet
    gsheet.write_to_google_sheet(worksheet_name, metrics_data)

def main() -> None:
  if len(sys.argv) != 7:
    raise Exception('Invalid arguments.')
  instance = sys.argv[1]
  start_time_sec = int(sys.argv[2])
  end_time_sec = int(sys.argv[3])
  period = int(sys.argv[4])
  test_type = sys.argv[5]
  worksheet_name = sys.argv[6]
  vm_metrics = VmMetrics()
  vm_metrics.fetch_metrics_and_write_to_google_sheet(start_time_sec,
                                                     end_time_sec, instance,
                                                     period, test_type,
                                                     worksheet_name)


if __name__ == '__main__':
  main()
