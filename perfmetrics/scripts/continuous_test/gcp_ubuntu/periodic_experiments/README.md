# GCSFuse Experiments Dashboard
GCSFuse Experiments Dashboard is a powerful tool designed to automate performance tests on GCSFuse, enabling developers to run tests with different configurations and visualize the results effectively. An experiment is a test run on GCSFuse for an experiment config.

## Introduction
GCSFuse Experiments Dashboard simplifies the process of running performance tests on GCSFuse by providing a user-friendly interface. It allows developers to define experiment configurations in a JSON file and execute tests with specific GCSFuse flags and branches effortlessly. The results from these tests are stored in BigQuery tables, and displayed on the GCSFuse Experiments Dashboard making it convenient for further analysis and comparison.

## Getting Started
To get started, follow the instructions in the subsequent sections to add or modify configurations, run experiments, and access the GCSFuse Experiments Dashboard for comprehensive result analysis.

## Adding or Modifying Configurations
To add or modify experiment configurations, developers need to update the JSON file located at perfmetrics/scripts/continuous_test/gcp_ubuntu/periodic_experiments/experiments_configuration.json. 

### Configuration Format:
```
{
    "experiment_configuration": [
        {
            "config_name": "<config_name>",
            "gcsfuse_flags": "<gcsfuse_flags>",
            "branch": "<branch>",
            "end_date": "<end_date>"
        },
        {
        ...
        },
        ...
    ]
}
```
### Key Descriptions: 
1. config_name (string): The name of the configuration. This should be a unique identifier for each experiment.
2. gcsfuse_flags (string): Flags to be passed to the gcsfuse command when running the experiment.
3. branch (string): The GCSFuse Repo branch to use for the experiment.
4. end_date (string): The experiment will run every day till this date and time. Format: "%Y-%m-%d %H:%M[:%S%:z]".

**Note**: Ensure that all components of the date (year, month, day, hour, minute, second) are entered as double digits.

### Configuration Example
#### An example of the configuration object in the experiment_configuration array:
```
"config_name": "TestConfiguration1"
"gcsfuse_flags": "--implicit-dirs --max-conns-per-host 100 --enable-storage-client-library --debug_fuse --debug_gcs --stackdriver-export-interval=30s"
"branch": "master"
"end_date": "2023-12-30 05:30:00+00:00"
```

### Uniquely Identifying an Experiment
##### A configuration name uniquely defines an experiment and two experiments can't have same configuration name. 
##### Once an experiment configuration has been defined, it is important to note that the gcsfuse flags and branch associated with that configuration cannot be edited directly. If there is a need to modify these values, the right approach is to create a new experiment configuration with the desired changes. This ensures the integrity and consistency of the existing configurations.

## Default Flags
Certain flags are set by default when running GCSFuse experiments. Notably, the --stackdriver-export-interval=30s flag is enabled by default to facilitate logging, which is always done in text format.

## Working with BigQuery
The test results obtained from the experiments are stored in BigQuery tables. The relevant project name, dataset ID, and table IDs can be modified in the constants.py file.

### All the python scripts and modules related to BigQuery are present at: perfmetrics/scripts/bigquery:
1. ‘setup.py’: This script is responsible for creating the dataset and tables in BigQuery. This file should be run once initialize the BigQuery environment.
2. ‘requirements.in’: This file contains the dependencies required by the Python scripts and modules for interacting with BigQuery. It lists the packages and their specific versions that must be installed for the code to work correctly. If no version is specified then the latest version will be installed.
3. ‘get_experiments_config.py’:This script is used to retrieve the configuration ID based on the provided configuration name, branch, gcsfuse flags, and end date.
4. ‘constants.py’: This file contains the IDs of the dataset, project and tables used in the Python scripts and modules for interacting with BigQuery By centralizing these IDs in one place, it makes it simpler to refer to them or manage and update them if needed.
5. ‘experiments_gcsfuse_bq.py’: This is the main Python module that contains methods for creating tables and exporting data to BigQuery.This module encapsulates the logic for working with BigQuery, providing functions that can be imported and used by other scripts
6. ‘experiments_gcsfuse_bq_test.py’: This file contains comprehensive unit tests for the experiments_gcsfuse_bq module. 

## Dashboard for Result Analysis
After the performance tests are completed, developers can access the GCSFuse Experiments Dashboard, which provides an interactive and insightful interface to compare and visualize the test results. The dashboard offers a range of visualization options, such as charts, and tables, enabling developers to identify performance patterns, analyze bottlenecks, and make data-driven decisions for optimizing GCSFuse.
