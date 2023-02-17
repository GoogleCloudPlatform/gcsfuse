# Python load testing tool

Python load testing tool is to generate load on the machine it is run using a
given task and then report the latencies of performing the task over the span
of load test.
A task is a piece of python code that is executed in multiple threads 
in multiple processes. Users can define their own tasks, pass them to the tool 
and pass different flags to configure the load testing.

Example usage:
```
python3 load_test.py --tasks-python-file-path /path/to/task/module.py 
--num-processes 32 --num-threads 2 --output-dir /dir/for/test/output 
--run-time 60
```
In the above example usage, two threads are spawned in each of the 32 processes
where each thread runs the task defined in /path/to/task/module.py in a
continuous loop till 60 seconds. After 60 seconds, the results containing 
latencies are saved in /dir/for/test/output.

# Prerequisites

* python3
* python packages mentioned in [requirements.txt]

[requirements.txt]: ./requirements.txt

# Supported flags
[load_test.py] is the script used that can be used to run load test. The script
accepts many flags to customize load testing configuration. Below are some 
important flags that can be passed to the script:
* ```--tasks-python-file-path```: Path to python module (file) containing task 
classes implementing task.LoadTestTask.
* ```--tasks-yaml-file-path```: Path to yaml file containing configurations for 
tasks. Note: Configurations in this file can only be of defined and recognised 
tasks. To know about the recognises types, see [sample_tasks.yaml].
* ```--num-processes```: Number of processes to spawn in load tests with
  --num-threads threads where each thread runs the task.
* ```--num-threads-per-process```: Number of threads to run in each process 
spawned for load test. Each thread runs the task in a loop depending and 
terminate depending upon other flags.
* ```--run-time```: Duration in seconds for which to run the load test.
* ```--output-dir```: Path to directory where you want to save the output of 
load tests. One file is created for each task with which load test is performed.

For more details on the supported flags, their default values and uses, please
run load_test.py script with ```--help``` flag.

[load_test.py]: ./load_test.py
[sample_tasks.yaml]: ./sample_tasks.yaml

# Output metrics
The output of load test contains the following metrics:
* General: Start time, end time, actual run time, tasks count.
* Latencies: Min, mean, max latencies and 25th, 50th, 95th and 99th percentiles 
of latencies of task performed over span of load test.

The output of load test performed using task with name SampleTask is saved
in the file ```output-dir/SampleTask.json```. 

# How to run

## Custom task
Let's say we want to run a CPU intensive task parallely with 40 processes for 5
minutes (300s) and save the result in file: ```~/output/CPUTask.json```

### Steps:
* Make sure the prerequisites are installed.
* Set ```PYTHONPATH = gcsfuse/perfmetrics/scripts/load_tests/python/```
* Create a module for task class and implement LoadTestTask class in it i.e.
define task method. E.g.
```
from load_generator import task

class CPUTask(task.LoadTestTask):
  
  def task(self, process_id, thread_id):
    s = 0
    for i in range(1000000):
      s = s + process_id * thread_id
    return s
```
cpu_task.py
* Run the following command:
```
python3 load_test.py --tasks-python-file-path cpu_task.py --num-processes 40 
--output-dir ~/output --run-time 300
```
* The latencies of CPU task performed over the span of load test is saved in 
```~/output/CPUTask.json```.

## Predefined tasks
The following [tasks] are predefined in tasks directory:
* python_os.py: Tasks to read files from disk python's native open api. Can be 
used with GCSFuse if disk is mounted using GCSFuse.
* tf_gfile.py: Tasks to read files from GCS using tf's tf.io.gfile.Gfile api. 
Can be used with GCSFuse or GCS files.
* tf_data.py: Tasks to read files from GCS using tf's tf.data api. Can be used 
with GCSFuse or GCS files.

For more details on the tasks, please refer to the module level 
description of files.

### Steps:
* Make sure the prerequisites are installed.
* Set ```PYTHONPATH = gcsfuse/perfmetrics/scripts/load_tests/python/```
* Create a yaml file containing configs for predefined tasks. E.g.
```
---
200mb_os:
 task_type: python_os_read
 file_path_format: ./gcs/200mb/read.{process_id}
 file_size: 200M
 
200mb_tf_data:
 task_type: tf_data_read
 file_path_format: gs://load-test-bucket-gcs/200mb/read.{file_num}.tfrecord
 file_size: 200M
 num_files: 3072
```
read_tasks.yaml. 

For more detials on the supported parameters in configs of predefined tasks, 
please refer to [sample_tasks.yaml]
* Run the following command:
```
python3 load_test.py --tasks-yaml-file-path read_tasks.yaml --num-processes 40 
--output-dir ~/output --run-time 300
```
* The latencies of read tasks performed over the span of load test is saved in
  ```~/output/200mb_os.json``` & ```~/output/200mb_tf_data.json```.

[sample_tasks.yaml]: ./sample_tasks.yaml
[tasks]: tasks

# Miscellaneous
* GCSFuse has to be mounted for using with [python_os.py] tasks.
* It is recommended to keep --num-processes and --num-threads as 1 for 
[tf_data.py] tasks as the parallelism is inside those tasks.
* All the tasks defined under [tasks] directory are marked as read/write tasks.
So, load_test.py script tries to create files before running actual load tests.
* Task using tf apis require gcloud login on machine.

[python_os.py]: tasks/python_os.py
[tf_data.py]: tasks/tf_data.py
[tasks]: tasks
