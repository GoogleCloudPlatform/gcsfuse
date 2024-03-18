# Experiment Configuration JSON File README

**NOTE**: If you modify a flag name within the development branch, ensure you also 
update the corresponding flag name within the [experiments_configuration.json](https://github.com/googlecloudplatform/gcsfuse/v2/blob/master/perfmetrics/scripts/continuous_test/gcp_ubuntu/periodic_experiments/experiments_configuration.json) 
file located in the master branch. This synchronization is crucial for running tests correctly.

## This JSON file contains configurations for different experiments.
### Configuration Format:
```
{
    "experiment_configuration": [
        {
            "config_name": "<config_name>",
            "gcsfuse_flags": "<gcsfuse_flags>",
            "branch": "<branch>",
            "config_file_flags_as_json":"<config_file_flag_value_in_json>",
            "end_date": "<end_date>"
        },
        {
        ...
        },
        ...
    ]
}
```
## Key Descriptions:
1. config_name (string): The name of the configuration. This should be a unique identifier for each experiment.
2. gcsfuse_flags (string): Flags to be passed to the gcsfuse command when running the experiment.
3. config_file_flags_as_json (json): --config-file flag to be passed to the gcsfuse command when running the experiment. (optional, default: null)
4. branch (string): The Git branch to use for the experiment.
4. end_date (string): The experiment will run every day till this date and time. Format: "YYYY-MM-DD HH:MM:SS".

## Configuration Example
### An example of the configuration object in the experiment_configuration array:
```
"config_name": "TestConfiguration1"
"gcsfuse_flags": "--implicit-dirs --max-conns-per-host 100 --enable-storage-client-library --debug_fuse --debug_gcs --stackdriver-export-interval=30s"
"branch": "master"
"config_file_flags_as_json": {
    "write": {
       "create-empty-file": true
    },
    "logging": {
          "file-path": "~/log.log",
          "format": "text",
          "severity": "info"
    }
 },
"end_date": "2023-12-30 05:30:00"
```

## Uniquely Identifying an Experiment
1. A configuration name uniquely defines an experiment and two experiments can't have same configuration name.
2. Once an experiment configuration has been defined, it is important to note that the gcsfuse flags and branch associated with that configuration cannot be edited directly. If there is a need to modify these values, the right approach is to create a new experiment configuration with the desired changes. This ensures the integrity and consistency of the existing configurations.
