## Experiment Configuration JSON File README
## This JSON file contains configurations for different experiments.

## Configuration Format:
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
## Key Descriptions: 
1. config_name (string): The name of the configuration. This should be a unique identifier for each experiment.
2. gcsfuse_flags (string): Flags to be passed to the gcsfuse command when running the experiment.
3. branch (string): The Git branch to use for the experiment.
4. end_date (string): The experiment will run every day till this date and time (time is optional). Format: "%Y-%m-%d[ %H:%M:%S%:z]".

**Note**: Ensure that all components of the date (year, month, day, hour, minute, second) are entered as double digits.

## Configuration Example
### An example of the configuration object in the experiment_configuration array:
```
"config_name": "TestConfiguration1"
"gcsfuse_flags": "--implicit-dirs --max-conns-per-host 100 --enable-storage-client-library --debug_fuse --debug_gcs --stackdriver-export-interval=30s"
"branch": "master"
"end_date": "2023-12-30 05:30:00+00:00"
```

## Uniquely Identifying an Experiment
#### A configuration name uniquely defines an experiment and two experiments can't have same configuration name. 
#### Once an experiment configuration has been defined, it is important to note that the gcsfuse flags and branch associated with that configuration cannot be edited directly. If there is a need to modify these values, the right approach is to create a new experiment configuration with the desired changes. This ensures the integrity and consistency of the existing configurations.
