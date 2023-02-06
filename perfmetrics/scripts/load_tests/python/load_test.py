"""Script to run load tests with given tasks and configuration.

The script requires tasks to run in load tests which can be written in a python
module and then path of that module can be passed to this load script using
--task-file-path flag.
Apart from the tasks, the load test can also be customized in multiple ways
like number of processes, threads inside each process, run time, total number
of tasks etc.
To know more about the flags supported in script, run:
  python3 load_test.py --help

Example:
  python3 load_test.py --task-file-path tasks/python_os.py --num-processes 40
  --num-threads 1 --run-time 90 --output-dir ~/output
"""
import logging
import importlib
import importlib.machinery
import importlib.util
import inspect
import os
import sys
import time
import argparse

from load_generator import load_generator as lg
from load_generator import task


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
  parser.add_argument(
      '--task-file-path',
      type=str,
      required=True,
      help='Path to python module (file) containing task classes implementing '
      'the base task.LoadTestTask')
  parser.add_argument(
      '--task-names',
      type=str,
      default='',
      help='Comma separated name(s) of tasks (task.LoadTestTask.TASK_NAME) in '
      'the --task-file-path. Load test is conducted only with those '
      'tasks. If empty string or nothing is passed then load test is '
      'conducted with all tasks.')
  parser.add_argument(
      '--output-dir',
      type=str,
      default='./output/',
      help='Path to directory where you want to save the output of load tests. '
      'One directory is created inside --output-dir for each task in '
      '--task-file-path file. If --only-print is also passed, then '
      'results are not saved.')
  parser.add_argument(
      '--num-processes',
      type=int,
      default=1,
      help='Number of processes to spawn in load tests with --num-threads '
      'threads where each thread runs the task.')
  parser.add_argument(
      '--num-threads',
      type=int,
      default=1,
      help='Number of threads to run in each process spawned for load test. '
      'Each thread runs the task in a loop depending and terminate '
      'depending upon other flags.')
  parser.add_argument(
      '--run-time',
      type=int,
      default=600,
      help='Duration in seconds for which to run the load test. Note: (a) The '
      f'load test runs for a minimum of {lg.MIN_OBSERVATION_INTERVAL_IN_SECS} '
      'seconds, irrespective of value passed. (b) The load test may terminate '
      'before depending upon value of --num-tasks and --num-tasks-per-thread '
      'flags passed.')
  parser.add_argument(
      '--num-tasks',
      type=int,
      default=sys.maxsize,
      help='Total number of times the given task to be performed across all '
      'threads in all processes of load test. Note: The actual number of '
      'times the task is performed may not be exactly equal but will be '
      'around the value passed.')
  parser.add_argument(
      '--num-tasks-per-thread',
      type=int,
      default=sys.maxsize,
      help='Total number of times the given task to be performed inside each '
      'thread of each process of load test. Note: The actual number of '
      'times the task is performed may not be exactly equal but will be '
      'around the value passed.')
  parser.add_argument(
      '--observation-interval',
      type=int,
      default=lg.MIN_OBSERVATION_INTERVAL_IN_SECS,
      help="Time interval in seconds, such that the resources' usages are "
      'observed after every given interval. E.g. if value is passed as 6, '
      "then the resources' usages are observed after every 6 seconds. "
      'Note: The minimum value that can be passed is '
      f'{lg.MIN_OBSERVATION_INTERVAL_IN_SECS} and the maximum value can be <= '
      'value of --run-time / 2')
  parser.add_argument(
      '--cooling-time',
      type=int,
      default=10,
      help='Time in seconds to wait after conducting load test on a task.')
  parser.add_argument(
      '--only-print',
      action='store_true',
      help='Whether to only print and not dump the results of load test on '
      'terminal.')
  parser.add_argument(
      '--log-level',
      type=str,
      default='INFO',
      help='Level of logging to print on terminal. Acceptable values are INFO '
      'and DEBUG.')
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


def dump_load_tests_metrics_into_csv(output_dir, load_test_results):
  """Dumps the common metrics of all load tests into a csv for comparison.

  Args:
    output_dir: String path to directory where metrics are to be dumped in csv.
    load_test_results: List of dictionary containing results of load tests.

  Returns:
    None
  """
  heading = [
      'S.No', 'Task Name', 'Avg. Download Bandwidth', 'Avg. Upload Bandwidth',
      'Avg. Bandwidth (computed)', 'Avg. CPU usage', 'Peak CPU usage',
      'Avg. Latency', 'Max. Latency'
  ]
  lines = [','.join(heading)]
  for idx, result in enumerate(load_test_results):
    line = [
        idx + 1,
        result[0],
        result[1]['avg_download_bw'],
        # avg_computed_net_bw is only available for read and write tasks.
        result[1]['avg_upload_bw'],
        result[1].get('avg_computed_net_bw', ''),
        result[1]['avg_cpu_usage'],
        result[1]['cpu_usage_pers']['max'],
        result[1]['task_lat_pers']['mean'],
        result[1]['task_lat_pers']['max']
    ]
    lines.append(','.join(list(map(str, line))) + '\n')

  with open(
      os.path.join(output_dir, 'comparison.csv'), 'w', encoding='UTF-8') as f_p:
    f_p.writelines(lines)


class LoadGeneratorForReadAndWriteTask(lg.LoadGenerator):
  """Custom load generator derived from load_generator.LoadGenerator.

  This load generator is specifically for read and write type of tasks. It can
  also be used for any other tasks but TASK_TYPE should not be set to read or
  write in those tasks.
  See base class for more details.
  """

  TASK_TYPES = ['read', 'write']

  def pre_load_test(self, **kwargs):
    """Pre-load test function to create files for read and write types of tasks.

    Only creates files using kwargs['task'].create_files method of size
    specified by kwargs['task'].FILE_SIZE when the task type is either read or
    write. Any other type of task can also be used but files won't be created
    for them.

    Args:
      kwargs: It is kept to provide flexibility to derived class implementation.

    Returns:
      None

    Raises:
      ValueError: When either FILE_PATH_FORMAT or FILE_SIZE is not set or empty
        in the task passed and task type is read or write.
      NotImplementedError: When create_files function is not implemented in the
        kwargs['task'].
    """
    # only run custom logic for read and write tasks
    if getattr(kwargs['task'], 'TASK_TYPE', '').lower() not in self.TASK_TYPES:
      return

    if (not hasattr(kwargs['task'], 'FILE_PATH_FORMAT')) or \
        (not hasattr(kwargs['task'], 'FILE_SIZE')):
      raise ValueError(
          'Task of types - read or write must have FILE_PATH_FORMAT'
          ' and FILE_SIZE (in bytes) attributes set.'
          'method set.')
    if not hasattr(kwargs['task'], 'create_files'):
      raise NotImplementedError(
          'Task of types - read or write must have create '
          'files function to create files before generating '
          'load.')

    file_path_format = getattr(kwargs['task'], 'FILE_PATH_FORMAT')
    file_size = getattr(kwargs['task'], 'FILE_SIZE')

    if (file_path_format == '') or (file_size == 0):
      raise ValueError("Constant FILE_PATH_FORMAT can't be empty and value"
                       "of FILE_SIZE can't be zero.")
    kwargs['task'].create_files(self.num_processes)

  def post_load_test(self,
                     observations,
                     output_dir='./',
                     dump_metrics=True,
                     print_metrics=True,
                     **kwargs):
    """Custom implementation of post-load test function.

    This function adds avg_computed_net_bw on top of default post load test
    implementation.
    avg_computed_net_bw = (SUM(values returned by task.LoadGenerator.task()) /
    actual_run_time).
    See base class implementation for more details.
    """
    metrics = super().post_load_test(observations, output_dir, dump_metrics,
                                     print_metrics)
    # only run custom logic for read and write tasks
    if getattr(kwargs['task'], 'TASK_TYPE', '').lower() not in self.TASK_TYPES:
      return metrics

    metrics = metrics['metrics']

    # compute bandwidth from task results
    total_io_bytes = sum(
        (task_result[4] for task_result in observations['tasks_results']))
    avg_computed_net_bw = total_io_bytes / metrics['actual_run_time']
    avg_computed_net_bw = avg_computed_net_bw / lg.MB
    metrics.update({'avg_computed_net_bw': avg_computed_net_bw})

    # dump metrics
    self._dump_metrics_into_json(metrics, output_dir)

    # print additional metrics
    print('\nNetwork bandwidth (computed by Sum(task response) / actual '
          'run time):')
    print('\tAvg. bandwidth (MiB/sec): ', metrics['avg_computed_net_bw'], '\n')
    return metrics


def main():
  args = parse_args()
  dump_metrics = not args.only_print

  logging.getLogger().setLevel(getattr(logging, args.log_level.upper()))

  logging.info('Initialising Load Generator...')
  lg_obj = LoadGeneratorForReadAndWriteTask(
      num_processes=args.num_processes,
      num_threads=args.num_threads,
      run_time=args.run_time,
      num_tasks_per_thread=args.num_tasks_per_thread,
      num_tasks=args.num_tasks,
      observation_interval=args.observation_interval)

  mod = import_module_using_src_code_path(args.task_file_path)
  mod_classes = []
  for _, cls in inspect.getmembers(mod, inspect.isclass):
    # Skip classes imported in the task file
    if cls.__module__ != mod.__name__:
      continue
    # Skip classes that are not of type task.LoadTestTask or don't
    # have implementation.
    if (not issubclass(cls, task.LoadTestTask)) or (inspect.isabstract(cls)):
      continue
    # Skip if user only wants to run for some specific tasks
    if len(args.task_names) and (cls.TASK_NAME not in args.task_names):
      continue
    mod_classes.append(cls)

  logging.info('Starting load generation...')
  load_test_results = []
  for idx, cls in enumerate(mod_classes):
    task_obj = cls()

    logging.info('\nRunning pre load test task for: %s', cls.TASK_NAME)
    lg_obj.pre_load_test(task=task_obj)

    logging.info('Generating load for: %s', cls.TASK_NAME)
    observations = lg_obj.generate_load(task_obj)

    output_dir = os.path.join(args.output_dir, cls.TASK_NAME)
    if dump_metrics and (not os.path.exists(output_dir)):
      os.makedirs(output_dir)

    logging.info('Running post load test task for: %s', cls.TASK_NAME)
    metrics = lg_obj.post_load_test(
        observations,
        output_dir=output_dir,
        dump_metrics=dump_metrics,
        print_metrics=True,
        task=task_obj)
    load_test_results.append((cls.TASK_NAME, metrics))
    logging.info('Load test completed for task: %s', cls.TASK_NAME)

    # don't sleep if it is last test.
    if idx < (len(mod_classes) - 1):
      logging.info('Sleeping for %s seconds...', args.cooling_time)
      time.sleep(args.cooling_time)

  # Create a comparison between all tasks and dump the csv.
  if (len(load_test_results) > 0) & dump_metrics:
    logging.info('Writing comparison of all tasks to csv...')
    dump_load_tests_metrics_into_csv(args.output_dir, load_test_results)


if __name__ == '__main__':
  main()
