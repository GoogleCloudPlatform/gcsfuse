# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Script to run load tests with given tasks and configuration.

The script requires tasks to run in load tests which can be written in a python
module or YAML file and then path of that file can be passed to this load script
using --tasks-python-file-path or --tasks-yaml-file path flag.
Apart from the tasks, the load test can also be customized in multiple ways
like number of processes, threads inside each process and run time.
To know more about the flags supported in script, run:
  python3 load_test.py --help

Example:
  python3 load_test.py --task-python-file-path tasks/python_os.py
  --num-processes 40 --num-threads 1 --run-time 90 --output-dir ~/output
"""
import logging
import importlib
import importlib.machinery
import importlib.util
import inspect
import sys
import os
import time
import argparse
import yaml
import re

from load_generator import load_generator as lg
from load_generator import task
from load_generator import constants as lg_const
from tasks import python_os
from tasks import tf_data
from tasks import tf_gfile

READ_WRITE_TASK_TYPES = [
    python_os.PYTHON_OS_READ, tf_data.TF_DATA_READ, tf_gfile.TF_GFILE_READ
]
KNOWN_TASK_TYPES = READ_WRITE_TASK_TYPES
SIZE_ABBREV_TO_SIZE = {'K': lg_const.KB, 'M': lg_const.MB, 'G': lg_const.GB}


def parse_args():
  """Parses the command line arguments and returns them.

  Returns:
    Arguments passed through command line set as attributes in args object.
  """
  parser = argparse.ArgumentParser(
      description='Script to do Load testing with '
      'given task using multiprocessing and '
      'multithreading on CPU and '
      'other resources.',
      formatter_class=argparse.ArgumentDefaultsHelpFormatter)
  group = parser.add_mutually_exclusive_group(required=True)
  group.add_argument(
      '--tasks-python-file-path',
      type=str,
      help='Path to python module (file) containing task classes implementing '
      'the base task.LoadTestTask')
  group.add_argument(
      '--tasks-yaml-file-path',
      type=str,
      help='Path to yaml file containing configs for known (predefined) '
      f'tasks: {KNOWN_TASK_TYPES}')
  parser.add_argument(
      '--task-names',
      type=str,
      default='',
      help='Comma separated name(s) of tasks (task.LoadTestTask.task_name) in '
      'the --tasks-python-file-path or --tasks-yaml-file-path. Load test is '
      'conducted only with those tasks. If empty string or nothing is passed '
      "then load test is conducted with all tasks. E.g. 'TaskA, TaskB', "
      "'TaskA,TaskB'.")
  parser.add_argument(
      '--output-dir',
      type=str,
      default=None,
      help='Path to directory where you want to save the output of load tests. '
  )
  parser.add_argument(
      '--num-processes',
      type=int,
      default=1,
      help='Number of processes to spawn in load tests with '
      '--num-threads-per-process threads where each thread runs the task.')
  parser.add_argument(
      '--num-threads-per-process',
      type=int,
      default=1,
      help='Number of threads to run in each process spawned for load test. '
      'Each thread runs the task in a loop and terminate depending upon other '
      'flags.')
  parser.add_argument(
      '--run-time',
      type=int,
      default=600,
      help='Duration in seconds for which to run the load test. Note: The load '
      'test may terminate before depending upon value of '
      '--num-executions-per-thread flags passed.')
  parser.add_argument(
      '--num-executions-per-thread',
      type=int,
      default=sys.maxsize,
      help='Total number of times the given task to be performed inside each '
      'thread of each process of load test. Note: It is possible that the task '
      'is not performed given number of times if --run-time is not enough.')
  parser.add_argument(
      '--start-delay',
      type=int,
      default=5,
      help='Time in seconds to wait before conducting load test on a task.')
  parser.add_argument(
      '--debug',
      action='store_true',
      help='Prints debug logs along with info logs.')
  args = parser.parse_args()
  args.task_names = args.task_names.replace(' ', '').split(',')
  args.task_names = [el for el in args.task_names if len(el)]
  return args


def import_module_using_src_code_path(src_code_path):
  """Imports the python module from its path to source code at runtime.

  Args:
    src_code_path: String path to source code of module.

  Returns:
    Imported python module
  """
  module_name = src_code_path.split('/')[-1].replace('.py', '')
  loader = importlib.machinery.SourceFileLoader(module_name, src_code_path)
  spec = importlib.util.spec_from_loader(loader.name, loader)
  mod = importlib.util.module_from_spec(spec)
  loader.exec_module(mod)
  return mod


def get_tasks_from_python_file_path(python_file_path):
  """Get tasks defined in given python module from its file path.

  Args:
    python_file_path: String file path to python module.

  Returns:
    List of task objects (task.LoadTestTask)
  """
  mod = import_module_using_src_code_path(python_file_path)
  task_objs = []
  for _, cls in inspect.getmembers(mod, inspect.isclass):
    # Skip classes imported in the task file
    if cls.__module__ != mod.__name__:
      continue
    # Skip classes that are not of type task.LoadTestTask or don't
    # have implementation.
    if (not issubclass(cls, task.LoadTestTask)) or (inspect.isabstract(cls)):
      continue
    task_objs.append(cls())
  return task_objs


def parse_file_size_str(file_size_str):
  """Gives file size in bytes from size string.

  Args:
    file_size_str: String file size.

  Returns:
    Size of file in bytes.

  Raises:
    ValueError: If file size string is not of recognised format.
  """
  if not bool(re.fullmatch(r'[0-9]+[kKmMgB]', file_size_str)):
    raise ValueError('The file size str set in config is not of recognised '
                     f'format: {file_size_str}')
  return int(file_size_str[:-1]) * \
         SIZE_ABBREV_TO_SIZE[file_size_str[-1].upper()]


def get_task_from_config(task_name, config):
  """Identify, creates and returns task from config defining task.

  Args:
     task_name: String name of task.
     config: Dictionary defining config for task.

  Returns:
    Task object instantiated using config.

  Raises:
    ValueError: If the task_type in config is not recognised and defined.
  """
  task_type = config['task_type']
  config.pop('task_type')
  config['file_size'] = parse_file_size_str(config['file_size'])
  if 'block_size' in config:
    config['block_size'] = parse_file_size_str(config['block_size'])
  task_cls = None
  if task_type == python_os.PYTHON_OS_READ:
    task_cls = python_os.OSRead
  elif task_type == tf_data.TF_DATA_READ:
    task_cls = tf_data.TFDataRead
  elif task_type == tf_gfile.TF_GFILE_READ:
    task_cls = tf_gfile.TFGFileRead
  else:
    raise ValueError(f'Given task type {task_type} is not in known types '
                     '{KNOWN_TASK_TYPES}')
  return task_cls(task_name=task_name, **config)


def get_tasks_from_yaml_file_path(yaml_file_path):
  """Gives tasks corresponding to configs in given yaml from its file path.

  Args:
    yaml_file_path: String path to yaml file.

  Returns:
    Task objects instantiated using configs defined in yaml file.
  """
  task_configs = {}
  with open(yaml_file_path, 'r', encoding='utf-8') as yaml_fh:
    task_configs = yaml.safe_load(yaml_fh)

  task_objs = []
  for task_name, config in task_configs.items():
    task_objs.append(get_task_from_config(task_name, config))
  return task_objs


class LoadGeneratorWithFileCreation(lg.LoadGenerator):
  """Custom load generator derived from load_generator.LoadGenerator.

  This implementation has pre- and post- load test methods for read and write
  files task types. (READ_WRITE_TASK_TYPES)
  See base class for more details.
  """

  def pre_load_generation(self, task_obj):
    """Pre- load test function to create files for read and write tasks.

    Calls task_obj.create_files method if task has create_files method
    implemented. create_files method is implemented in case of
    READ_WRITE_TASK_TYPES.

    Args:
      task_obj: Task object on which load testing to be performed.

    Returns:
      None
    """
    if getattr(task_obj, 'task_type', '') in READ_WRITE_TASK_TYPES and \
        hasattr(task_obj, 'create_files'):
      task_obj.create_files(self.num_processes)

  def post_load_generation(self, observations, output_file, print_metrics,
                           task_obj):
    """Custom implementation of post-load test function.

    This function adds avg_computed_net_bw on top of default post load test
    implementation.
    avg_computed_net_bw = (SUM(values returned by task.LoadGenerator.task()) /
    actual_run_time).
    See base class implementation for more details.
    """
    metrics = super().post_load_generation(observations, output_file,
                                           print_metrics)
    # only run custom logic for read and write tasks
    if getattr(task_obj, 'task_type', '') not in READ_WRITE_TASK_TYPES:
      return metrics

    # compute bandwidth from task results
    total_io_bytes = sum(
        (task_result.result
         for task_result in observations[lg_const.TASKS_RESULTS]))
    avg_computed_net_bw = total_io_bytes / metrics[lg_const.ACTUAL_RUN_TIME]
    avg_computed_net_bw = avg_computed_net_bw / lg_const.MB
    metrics.update({'avg_computed_net_bw': avg_computed_net_bw})

    # Re-dump the metrics to same file.
    if output_file:
      self._dump_metrics_into_json(metrics, output_file)

    if print_metrics:
      # print additional metrics
      print('\nNetwork bandwidth (computed by Sum(task response) / actual '
            'run time):')
      print('\tAvg. bandwidth (MiB/sec): ', metrics['avg_computed_net_bw'],
            '\n')
    return metrics


def main():
  args = parse_args()

  logging.getLogger().setLevel(logging.INFO)
  if args.debug:
    logging.getLogger().setLevel(logging.DEBUG)

  logging.info('Initialising Load Generator...')
  lg_obj = LoadGeneratorWithFileCreation(
      num_processes=args.num_processes,
      num_threads_per_process=args.num_threads_per_process,
      run_time=args.run_time,
      num_executions_per_thread=args.num_executions_per_thread)

  task_objs = []
  if args.tasks_python_file_path:
    task_objs = get_tasks_from_python_file_path(args.tasks_python_file_path)
  else:
    task_objs = get_tasks_from_yaml_file_path(args.tasks_yaml_file_path)

  # keep only those tasks passed in args.task_names. Note: If task_names = ''
  # then all tasks are kept.
  filtered_task_objs = []
  for task_obj in task_objs:
    if len(args.task_names) == 0 or task_obj.task_name in args.task_names:
      filtered_task_objs.append(task_obj)

  logging.info('Starting load generation...')
  load_test_results = []
  for task_obj in filtered_task_objs:
    logging.info('\nSleeping for: %s seconds', args.start_delay)
    time.sleep(args.start_delay)

    logging.info('\nRunning pre load test task for: %s', task_obj.task_name)
    lg_obj.pre_load_generation(task_obj=task_obj)

    logging.info('Generating load for: %s', task_obj.task_name)
    observations = lg_obj.generate_load(task_obj)

    output_file = None
    if args.output_dir and (not os.path.exists(args.output_dir)):
      os.makedirs(args.output_dir)
      output_file = os.path.join(args.output_dir, f'{task_obj.task_name}.json')

    logging.info('Running post load test task for: %s', task_obj.task_name)
    metrics = lg_obj.post_load_generation(
        observations,
        output_file=output_file,
        print_metrics=True,
        task_obj=task_obj)
    load_test_results.append((task_obj.task_name, metrics))
    logging.info('Load test completed for task: %s', task_obj.task_name)


if __name__ == '__main__':
  main()
