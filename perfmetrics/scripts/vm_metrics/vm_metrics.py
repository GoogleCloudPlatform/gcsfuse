"""Extracts required Google Cloud metrics using Monitoring V3 API call, parses
   the API response into a list and writes to google sheet.

   Takes VM instance name, interval start time, interval end time, alignment
   period, fio test type and google sheet name as command line inputs.
   The supported fio test types are: read, write
   Metrics extracted:
   1.Peak Cpu Utilization(%)
   2.Mean Cpu Utilization(%)
   3.Peak Network Bandwidth(By/s)
   4.Mean Network Bandwidth(By/s)
   5.Opencensus Error Count
   6.Opencensus Mean Latency(s)
   7.Read Bytes Count(By)

  Usage:
  >>python3 vm_metrics.py {instance} {start time in epoch sec} {end time in epoch sec} {period in sec} {test_type} {worksheet_name}

"""
import dataclasses
import os
import sys
import google.api_core
from google.api_core.exceptions import GoogleAPICallError
import google.cloud
from google.cloud import monitoring_v3
from gsheet import gsheet

PROJECT_NAME = 'projects/gcs-fuse-test'

CPU_UTI_METRIC = 'compute.googleapis.com/instance/cpu/utilization'
RECEIVED_BYTES_COUNT_METRIC = 'compute.googleapis.com/instance/network/received_bytes_count'
OPS_ERROR_COUNT_METRIC = 'custom.googleapis.com/gcsfuse/fs/ops_error_count'
OPS_LATENCY_METRIC = 'custom.googleapis.com/gcsfuse/fs/ops_latency'
READ_BYTES_COUNT_METRIC = 'custom.googleapis.com/gcsfuse/gcs/read_bytes_count'

ERROR_COUNT_FILTERED_OUT_FS_OP = ['GetXattr']

class NoValuesError(Exception):
  """API response values are missing."""


@dataclasses.dataclass
class MetricPoint:
  value: float
  start_time_sec: int
  end_time_sec: int


def _get_ops_metric_method_name(test_type):
  """Getting the ops metric fs_op name from the test type.
  """
  if test_type == 'read' or test_type == 'randread':
    return 'ReadFile'
  elif test_type == 'write' or test_type == 'randwrite':
    return 'WriteFile'

def _parse_metric_value_by_type(value, value_type) -> float:
  """Parses the value from a value object in API response.

    Args:
      value (object): The value object from API response
      value_type (int) : Integer representing the value type of the object
  """
  if value_type == 3:
    return value.double_value
  elif value_type == 2:
    return value.int64_value
  elif value_type == 5:
    return value.distribution_value.mean
  else:
    raise Exception('Unhandled Value type')

def _get_metric_filter(type, metric_type, instance):
  """Getting the metrics filter from metric type and instance name.

    Args:
      metric_type (str): The type of metric
      instance (str): VM instance name
  """
  if(type == 'compute'):
    return ('metric.type = "{metric_type}" AND metric.label.instance_name '
          '={instance_name}').format(metric_type=metric_type, instance_name=instance)
  elif(type == 'custom'):
    return ('metric.type = "{metric_type}" AND metric.labels.opencensus_task = '
          'ends_with("{instance_name}")').format(metric_type=metric_type, instance_name=instance)
  else:
    raise Exception('Unhandled metric type')

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
                                 point.interval.start_time.seconds,
                                 point.interval.end_time.seconds)

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

  def _get_api_response(self, metric_type, start_time_sec, end_time_sec,
                        instance, period, aligner, test_type, reducer, group_fields):
    """Fetches the API response for the requested metrics.

    Args:
      metric_type (str): The type of metric
      start_time_sec (int): Epoch seconds
      end_time_sec (int): Epoch seconds
      instance (str): VM instance name
      period (float): Period over which the values are aligned
      aligner(str): Operation to be applied at points of each period
      test_type(str): The type of load test for which metrics are taken 
      reducer(str): Operation to aggregate data points accross multiple metrics
      group_fields(list[str]): The fields we want to aggregate using reducer
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
        per_series_aligner=getattr(monitoring_v3.Aggregation.Aligner, aligner),
        cross_series_reducer=getattr(monitoring_v3.Aggregation.Reducer,reducer),
        group_by_fields=group_fields
    )

    # Checking whether the metric is custom or compute by getting the first 6 or 7 elements of metric type:
    if(metric_type[0:7]=='compute'):
      metric_filter = _get_metric_filter('compute', metric_type, instance)

    elif (metric_type[0:6] == 'custom'):
      metric_filter = _get_metric_filter('custom', metric_type, instance)

      # Adding extra filters:
      if (metric_type == OPS_ERROR_COUNT_METRIC):
        for fs_op in ERROR_COUNT_FILTERED_OUT_FS_OP:
          metric_filter = ('{} AND metric.labels.fs_op != {}').format(metric_filter, fs_op)
      elif (metric_type == OPS_LATENCY_METRIC):
        fs_type = _get_ops_metric_method_name(test_type)
        metric_filter = ('{} AND metric.labels.fs_op = {}').format(metric_filter, fs_type)
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
                                ).format(metric_type))

    return metrics_response

  def _get_metrics(self, start_time_sec, end_time_sec, instance, period,
                   metric_type, factor, aligner, test_type, reducer='REDUCE_NONE',
                   group_fields=['metric.type']):
    """Returns the MetricPoint list for requested metric type.

    Args:
      start_time_sec (int): Epoch seconds
      end_time_sec (int): Epoch seconds
      instance (str): VM instance name
      period (float): Period over which the values are aligned
      metric_type (str): The type of metric
      factor (float) : Converting the API response values into required units.
      aligner(str): Operation to be applied at points of each period
      test_type(str): The type of load test for which metrics are taken
      reducer(str): Operation to aggregate data points accross multiple metrics
      group_fields(list[str]): The fields we want to aggregate using reducer
    Returns:
      list[MetricPoint]
    """
    metrics_response = self._get_api_response(metric_type, start_time_sec,
                                              end_time_sec, instance, period,
                                              aligner, test_type, reducer, group_fields)
    metrics_data = _create_metric_points_from_response(metrics_response, factor)
    
    # In case OPS_ERROR_COUNT data is empty, we return a list of zeroes:
    if(metric_type == OPS_ERROR_COUNT_METRIC and metrics_data == []):
      return [MetricPoint(0, 0, 0) for i in range(int((end_time_sec-start_time_sec)/period)+1)]
    
    # Metrics data for metrics other that OPS_ERROR_COUNT_DATA should not be empty:
    if(metrics_data == []):
      raise NoValuesError('No values were retrieved from the call')

    return metrics_data

  def fetch_metrics_and_write_to_google_sheet(self, start_time_sec,
                                              end_time_sec, instance,
                                              period, test_type, worksheet_name) -> None:
    """Fetches the metrics data for all types and writes it to a google sheet.

    Args:
      start_time_sec (int): Epoch seconds
      end_time_sec (int): Epoch seconds
      instance (str): VM instance
      period (float): Period over which the values are taken
      test_type(str): The type of load test for which metrics are taken
      worksheet_name(str): The name of the worksheet of the google sheet we ware writing into
    Returns: None
    """
    self._validate_start_end_times(start_time_sec, end_time_sec)

    cpu_uti_peak_data = self._get_metrics(start_time_sec, end_time_sec,
                                          instance, period, CPU_UTI_METRIC,
                                          1 / 100, 'ALIGN_MAX', test_type)
    cpu_uti_mean_data = self._get_metrics(start_time_sec, end_time_sec,
                                          instance, period, CPU_UTI_METRIC,
                                          1 / 100, 'ALIGN_MEAN', test_type)
    rec_bytes_peak_data = self._get_metrics(start_time_sec, end_time_sec,
                                            instance, period,
                                            RECEIVED_BYTES_COUNT_METRIC, 60,
                                            'ALIGN_MAX', test_type)
    rec_bytes_mean_data = self._get_metrics(start_time_sec, end_time_sec,
                                            instance, period,
                                            RECEIVED_BYTES_COUNT_METRIC, 60,
                                            'ALIGN_MEAN', test_type)
    ops_latency_mean_data = self._get_metrics(start_time_sec, end_time_sec,
                                              instance, period,
                                              OPS_LATENCY_METRIC, 1,
                                              'ALIGN_DELTA', test_type)
    read_bytes_count_data = self._get_metrics(start_time_sec, end_time_sec,
                                              instance, period,
                                              READ_BYTES_COUNT_METRIC, 1,
                                              'ALIGN_DELTA', test_type)
    ops_error_count_data = self._get_metrics(start_time_sec, end_time_sec,
                                             instance, period,
                                             OPS_ERROR_COUNT_METRIC, 1,
                                             'ALIGN_DELTA',test_type, 'REDUCE_SUM',
                                             ['metric.labels'])

    metrics_data = []
    for cpu_uti_peak, cpu_uti_mean, rec_bytes_peak, rec_bytes_mean, ops_latency, read_bytes_count, ops_error_count in zip(
        cpu_uti_peak_data, cpu_uti_mean_data, rec_bytes_peak_data,
        rec_bytes_mean_data, ops_latency_mean_data, read_bytes_count_data,
        ops_error_count_data):
      metrics_data.append([
          cpu_uti_peak.start_time_sec, cpu_uti_peak.value, cpu_uti_mean.value,
          rec_bytes_peak.value, rec_bytes_mean.value,
          ops_latency.value, read_bytes_count.value, ops_error_count.value
      ])

    # Writing metrics data to google sheet
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
                                                     period, test_type, worksheet_name)

if __name__ == '__main__':
  main()
