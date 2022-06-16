"""
To execute the script:
>>blaze run vmmetrics {instance} {start time in epoch sec} {end time in epoch sec} {mean period in sec}

The code takes input the start time and end time (in epoch seconds) and the
instance name and the mean period. Then it creates an instance of VmMetrics
class and calls its methods with all the corresponding parameters, makes the api
call, reads the response and returns a list of metric points where each point
has the peak value and mean value of a mean period. Then it dumps all the data
into a google sheet.

"""
import sys
import os
import time
import datetime
import dataclasses
import google.api_core
import google.cloud
from google.api_core.exceptions import GoogleAPICallError
from google.cloud import monitoring_v3
sys.path.append("./gsheet")
import gsheet

WORKSHEET_NAME = 'vm_metrics!'

PROJECT_NAME = 'projects/gcs-fuse-test'
CPU_UTI_METRIC = 'compute.googleapis.com/instance/cpu/utilization'
RECEIVED_BYTES_COUNT_METRIC = 'compute.googleapis.com/instance/network/received_bytes_count'


class NoValuesError(Exception):
  """API response values are missing."""

@dataclasses.dataclass
class MetricPoint:
  peak_value: float 
  mean_value: float
  start_time_sec: int
  end_time_sec: int

def _parse_metric_value_by_type(peak_value, peak_value_type) -> float:
  if peak_value_type == 3:
    return peak_value.double_value
  elif peak_value_type == 2:
    return peak_value.int64_value
  else:
    raise Exception('Unhandled Value type')


def _create_metric_points_from_response(peak_metrics_response,
                                        mean_metrics_response,
                                        factor):
  """Parses the given peak and mean metrics and returns a list of MetricPoint.

    Args:
      peak_metrics_response (object): The peak metrics API response
      mean_metrics_response (object): The mean metrics API response
      factor (float) : For converting the API response values into appropriate units.

    Returns:
      list[MetricPoint]

    Raises:
      NoValuesError: Raise when API response is empty.

  """
  metric_point_list = []
  for peak_metric, mean_metric in zip(peak_metrics_response,
                                      mean_metrics_response):
    for point_peak, point_mean in zip(peak_metric.points, mean_metric.points):
      peak_value = _parse_metric_value_by_type(point_peak.value,
                                               peak_metric.value_type)
      metric_point = MetricPoint(peak_value / factor,
                                 point_mean.value.double_value / factor,
                                 point_peak.interval.start_time.seconds,
                                 point_peak.interval.end_time.seconds)

      metric_point_list.append(metric_point)

  if len(metric_point_list) == 0:
    raise NoValuesError('No values were retrieved from the call')
  metric_point_list.reverse()
  return metric_point_list


class VmMetrics:

  def _validate_start_end_times(self, start_time_sec, end_time_sec):
    """Checks that start time is less than end time.

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
                        instance, period):
    """Fetches the API response for peak and mean metrics.

    Args:
      metric_type (str): The type of metric
      start_time_sec (int): Epoch seconds
      end_time_sec (int): Epoch seconds
      instance (str): VM instance name
      period (float): Period over which the mean and peak values are taken
    Returns:
      list[peak metrics API response, mean metrics API response]
    Raises:
      GoogleAPICallError
    """

    client = monitoring_v3.MetricServiceClient()
    interval = monitoring_v3.TimeInterval(
        end_time={'seconds': int(end_time_sec)},
        start_time={'seconds': int(start_time_sec)})

    aggregation_peak = monitoring_v3.Aggregation(
        alignment_period={'seconds': period},
        per_series_aligner=monitoring_v3.Aggregation.Aligner
        .ALIGN_MAX,
    )

    aggregation_mean = monitoring_v3.Aggregation(
        alignment_period={'seconds': period},
        per_series_aligner=monitoring_v3.Aggregation.Aligner
        .ALIGN_MEAN,
    )

    metric_filter = ('metric.type = "{metric_type}" AND '
                     'metric.label.instance_name ={instance_name}').format(
                         metric_type=metric_type, instance_name=instance)
    try:
      peak_metrics_response = client.list_time_series({
          'name':PROJECT_NAME,
          'filter':metric_filter,
          'interval':interval,
          'view':monitoring_v3.ListTimeSeriesRequest
          .TimeSeriesView.FULL,
          'aggregation':aggregation_peak,
      })
    except:
      raise GoogleAPICallError('The request for peak response of ' +
                               metric_type + ' failed, Please try again.')

    try:
      mean_metrics_response = client.list_time_series({
          'name':PROJECT_NAME,
          'filter':metric_filter,
          'interval':interval,
          'view':monitoring_v3.ListTimeSeriesRequest
          .TimeSeriesView.FULL,
          'aggregation':aggregation_mean,
      })
    except:
      raise GoogleAPICallError('The request for mean response of ' +
                               metric_type + ' failed, Please try again.')

    return [peak_metrics_response, mean_metrics_response]

  def _get_metrics(self, start_time_sec, end_time_sec, instance, period,
                   metric_type, factor):
    """Returns the metrics data for requested metric type.

    Args:
      start_time_sec (int): Epoch seconds
      end_time_sec (int): Epoch seconds
      instance (str): VM instance
      period (float): Period over which the mean and peak values are taken
      metric_type (str): The metric whose data is to be retrieved
      factor (float) : The factor by which the value of API response should be
                       divided to get inot desired units.
    Returns:
      list[MetricPoint]
    """
    peak_metrics_response, mean_metrics_response = self._get_api_response(
        metric_type, start_time_sec, end_time_sec, instance, period)
    metrics_data = _create_metric_points_from_response(
        peak_metrics_response, mean_metrics_response, factor)

    return metrics_data

  def fetch_metrics_and_write_to_google_sheet(self, start_time_sec,
                                              end_time_sec, instance,
                                              period) -> None:
    """Fetches the metrics data for cpu utilization and received bytes count and writes it to a google sheet.

    Args:
      start_time_sec (int): Epoch seconds
      end_time_sec (int): Epoch seconds
      instance (str): VM instance
      period (float): Period over which the mean and peak values are taken
    Returns: None
    """
    self._validate_start_end_times(start_time_sec, end_time_sec)
    cpu_uti_data = self._get_metrics(start_time_sec, end_time_sec, instance,
                                     period, CPU_UTI_METRIC, 1 / 100)
    rec_bytes_data = self._get_metrics(start_time_sec, end_time_sec, instance,
                                       period, RECEIVED_BYTES_COUNT_METRIC,
                                       60000)
    metrics_data = []
    for cpu_uti_metric_point, rec_bytes_metric_point in zip(
        cpu_uti_data, rec_bytes_data):
      metrics_data.append([
          cpu_uti_metric_point.start_time_sec, cpu_uti_metric_point.peak_value,
          cpu_uti_metric_point.mean_value, rec_bytes_metric_point.peak_value,
          rec_bytes_metric_point.mean_value
      ])

    # writing metrics data to google sheet
    gsheet.write_to_google_sheet(WORKSHEET_NAME, metrics_data)


def main() -> None:
  if len(sys.argv) != 5:
    raise Exception('Invalid arguments.')
  instance = sys.argv[1]
  start_time_sec = int(sys.argv[2])
  end_time_sec = int(sys.argv[3])
  period = int(sys.argv[4])
  vm_metrics = VmMetrics()
  vm_metrics.fetch_metrics_and_write_to_google_sheet(start_time_sec,
                                                     end_time_sec, instance,
                                                     period)

if __name__ == '__main__':
  main()
