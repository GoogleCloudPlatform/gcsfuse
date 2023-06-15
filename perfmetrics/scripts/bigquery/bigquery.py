import argparse
from google.cloud import bigquery
import sys

PROJECT_ID = 'gcs-fuse-test'
DATASET_ID = 'performance_metrics'
CONFIGURATION_TABLE_ID = 'experiment_configuration'
FIO_TABLE_ID = 'fio_metrics'
VM_TABLE_ID = 'vm_metrics'
LS_TABLE_ID = 'ls_metrics'

# Construct a BigQuery client object.
client = bigquery.Client()

class BigQuery():

  def insert_config_and_get_config_id(self, gcsfuse_flags, branch, end_date) -> int:

    """Gets the experiment configuration and checks if it is already present in the BigQuery tables,
       If not present: New configuration will be inserted, and it's configuration id will be returned
       If present: Configuration id will be fetched and returned

    Args:
      gcsfuse_flags: GCSFuse flags for mounting the test buckets.
      branch: GCSFuse repo branch to be used for building GCSFuse.
      end_date: Date upto when tests are run.

    Return:
      Configuration id of the experiment in the BigQuery tables
    """
    dataset_ref = None
    try:
      dataset_ref = client.dataset(DATASET_ID)
    except:
      dataset_ref = client.create_dataset(DATASET_ID)

    # Query for creating experiment_configuration table if it does not exist
    query_create_table_experiment_configuration = """
        CREATE TABLE IF NOT EXISTS {}.{}.{}(
          configuration_id INT64,
          gcsfuse_flags STRING,
          branch STRING,
          end_date TIMESTAMP,
          PRIMARY KEY (configuration_id) NOT ENFORCED
        ) OPTIONS (description = 'Table for storing Job Configurations and respective VM instance name on which the job was run');
    """.format(PROJECT_ID, DATASET_ID, CONFIGURATION_TABLE_ID)

    # API Request to create experiment_configuration if it does not exist
    results = client.query(query_create_table_experiment_configuration)
    print(results)

    query_get_configuration_id = """
      SELECT configuration_id
      FROM `{}.{}.{}`
      WHERE gcsfuse_flags = '{}'
      AND branch = '{}'
      AND end_date = '{}'
    """.format(PROJECT_ID, DATASET_ID, CONFIGURATION_TABLE_ID, gcsfuse_flags, branch, end_date)

    config_id = None
    exists = False
    query_job = client.query(query_get_configuration_id)
    for row in query_job:
      config_id = row['configuration_id']
      exists = True
    if not exists:
      table_ref = dataset_ref.table(CONFIGURATION_TABLE_ID)
      table = client.get_table(table_ref)
      row_count = table.num_rows
      config_id = row_count + 1
      rows_to_insert = [(config_id, gcsfuse_flags, branch, end_date)]
      errors = client.insert_rows(table, rows_to_insert)
      print(errors)

    return config_id

  def setup_bigquery(self):

    """Creates the tables to store the metrics data if they don't exist in the dataset
    """
    # Query for creating fio_metrics table
    query_create_table_fio_metrics = """
        CREATE TABLE IF NOT EXISTS {}.{}.{}(
          configuration_id INT64, 
          start_time_build TIMESTAMP,
          test_type STRING, 
          num_threads INT64, 
          file_size_kb INT64, 
          block_size_kb INT64,
          start_time INT64, 
          end_time INT64, 
          iops FLOAT64, 
          bandwidth_bytes_per_sec INT64, 
          IO_bytes INT64, 
          min_latency FLOAT64, 
          max_latency FLOAT64, 
          mean_latency FLOAT64, 
          percentile_latency_20 FLOAT64, 
          percentile_latency_50 FLOAT64, 
          percentile_latency_90 FLOAT64, 
          percentile_latency_95 FLOAT64, 
          FOREIGN KEY(configuration_id) REFERENCES {}.{} (configuration_id) NOT ENFORCED
        ) OPTIONS (description = 'Table for storing FIO metrics extracted from periodic performance load testing');
    """.format(PROJECT_ID, DATASET_ID, FIO_TABLE_ID, DATASET_ID, CONFIGURATION_TABLE_ID)

    # Query for creating vm_metrics table
    query_create_table_vm_metrics = """
        CREATE TABLE IF NOT EXISTS {}.{}.{}(
          configuration_id INT64, 
          start_time_build TIMESTAMP,
          end_time INT64, 
          cpu_utilization_peak_percentage FLOAT64, 
          cpu_utilization_mean_percentage FLOAT64, 
          received_bytes_peak_bytes_per_sec FLOAT64, 
          received_bytes_mean_bytes_per_sec FLOAT64, 
          read_bytes_count INT64,
          ops_error_count INT64, 
          ops_mean_latency_sec FLOAT64, 
          sent_bytes_per_sec FLOAT64, 
          memory_utilization_ram FLOAT64,
          memory_utilization_disk_tempdir FLOAT64,
          iops FLOAT64, 
          ops_count_list_object INT64, 
          ops_count_create_object INT64, 
          ops_count_stat_object INT64, 
          ops_count_new_reader INT64, 
          FOREIGN KEY(configuration_id) REFERENCES {}.{} (configuration_id) NOT ENFORCED
        ) OPTIONS (description = 'Table for storing VM metrics extracted from periodic performance load testing');
    """.format(PROJECT_ID, DATASET_ID, VM_TABLE_ID, DATASET_ID, CONFIGURATION_TABLE_ID)

    # Query for creating ls_metrics table
    query_create_table_ls_metrics = """
        CREATE TABLE IF NOT EXISTS {}.{}.{}(
          configuration_id INT64,
          start_time_build TIMESTAMP,
          test_type STRING, 
          command STRING,
          start_time INT64, 
          end_time INT64,
          num_files INT64, 
          num_samples INT64, 
          min_latency_msec FLOAT64,
          max_latency_msec FLOAT64,
          mean_latency_msec FLOAT64, 
          median_latency_msec FLOAT64, 
          standard_dev_msec FLOAT64, 
          percentile_latency_20 FLOAT64, 
          percentile_latency_50 FLOAT64, 
          percentile_latency_90 FLOAT64, 
          percentile_latency_95 FLOAT64, 
          cpu_utilization_peak_percentage FLOAT64, 
          cpu_utilization_mean_percentage FLOAT64,
          memory_utilization_ram FLOAT64, 
          FOREIGN KEY(configuration_id) REFERENCES {}.{} (configuration_id) NOT ENFORCED
        ) OPTIONS (description = 'Table for storing GCSFUSE metrics extracted from listing benchmark tests');
    """.format(PROJECT_ID, DATASET_ID, LS_TABLE_ID, DATASET_ID, CONFIGURATION_TABLE_ID)

    # API Requests
    results = client.query(query_create_table_fio_metrics)
    print(results)
    results = client.query(query_create_table_vm_metrics)
    print(results)
    results = client.query(query_create_table_ls_metrics)
    print(results)


def _parse_arguments(argv):
  """Parses the arguments provided to the script via command line.

  Args:
    argv: List of arguments received by the script.

  Returns:
    A class containing the parsed arguments.
  """
  argv = sys.argv
  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--gcsfuse_flags',
      help='GCSFuse flags for mounting the test buckets',
      action='store',
      nargs=1,
      required=True
  )
  parser.add_argument(
      '--branch',
      help='GCSFuse repo branch to be used for building GCSFuse.',
      action='store',
      nargs=1,
      required=True
  )
  parser.add_argument(
      '--end_date',
      help='Date upto when tests are run',
      action='store',
      nargs=1,
      required=True
  )
  return parser.parse_args(argv[1:])

if __name__ == '__main__':
  argv = sys.argv
  args = _parse_arguments(argv)
  bigquery_obj = BigQuery()
  config_id = bigquery_obj.insert_config_and_get_config_id(args.gcsfuse_flags[0], args.branch[0], args.end_date[0])
  print(config_id)