from google.cloud import bigquery

PROJECT_ID = 'gcsfuse-intern-project-2023'
DATASET_ID = 'performance_metrics'
CONFIGURATION_TABLE_ID = 'experiment_configuration'
FIO_TABLE_ID = 'fio_metrics'
VM_TABLE_ID = 'vm_metrics'
LS_TABLE_ID = 'ls_metrics'

class BigQuery():

  def setup_bigquery(self):

    # Construct a BigQuery client object.
    client = bigquery.Client()

    # Query for creating fio_metrics table
    query_create_table_fio_metrics = """
        CREATE OR REPLACE TABLE {}.{}.{}(
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
        CREATE OR REPLACE TABLE {}.{}.{}(
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
        CREATE OR REPLACE TABLE {}.{}.{}(
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

    # Executing the queries
    results = client.query(query_create_table_fio_metrics)
    print(results)
    results = client.query(query_create_table_vm_metrics)
    print(results)
    results = client.query(query_create_table_ls_metrics)
    print(results)
