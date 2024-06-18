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
"""Load Generator for performing load test with a given task (in python).

Contains the class for generating load with the help of a task written in
python.

Example:

    task_obj = ReadTask()
    lg_obj = LoadGenerator(...)
    lg_obj.pre_load_generation(...)
    observations = lg_obj.generate_load(task_obj)
    metrics = lg_obj.post_load_generation(observations, ...)
"""
import logging
import sys
import threading
import time
import json
import multiprocessing
import numpy as np
from dataclasses import dataclass
from typing import Any
from load_generator import constants as lg_const


@dataclass(frozen=True)
class TaskExecutionResult:
  """Dataclass for a single task (pre, task & post) execution result.

  process_id: Integer value denoting the process id in which the task was run.
  thread_id: Integer value denoting the thread id in process_id in which the
    task was run.
  start_time: Float value denoting the time when the task was started.
  end_time: Float value denoting the time when the task was ended.
  result: Any type of value that the task returns
  """
  process_id: int
  thread_id: int
  start_time: float
  end_time: float
  result: Any


class LoadGenerator:
  """Generates load using a given task.

  Generates load on CPU and other resources depending upon the given task. This
  class also provides default implementation of post load test task to get 
  latencies of tasks. Classes derived from this class can define
  their own pre- and post-load test tasks and run before and after actual load
  test.

  Args:
    num_processes: An integer defining the number of processes to run during
      load test.
    num_threads_per_process: An integer defining the number of threads to run
      in each process during load test.
    run_time: An integer defining the number of seconds to run the load test
      for.
    num_executions_per_thread: An integer defining the number of times the given
      task to be run inside each thread of each process during load test.
  """

  def __init__(self,
               num_processes,
               num_threads_per_process,
               run_time=sys.maxsize,
               num_executions_per_thread=sys.maxsize):
    self.num_processes = num_processes
    self.num_threads_per_process = num_threads_per_process
    self.run_time = min(sys.maxsize, run_time)
    self.num_executions_per_thread = min(sys.maxsize, num_executions_per_thread)

    if (self.run_time == sys.maxsize) & \
        (self.num_executions_per_thread == sys.maxsize):
      raise ValueError('Out of run_time and num_executions_per_thread '
                       'arguments, one has to be passed.')

    self.total_num_tasks = sys.maxsize
    if self.num_executions_per_thread != sys.maxsize:
      self.total_num_tasks = self.num_executions_per_thread * \
                             self.num_threads_per_process * self.num_processes

  def pre_load_generation(self):
    """Task to perform before running load test.
    """
    pass

  def generate_load(self, task):
    """Performs load test using the given task.

    The load is generated on CPU and other resources (depending upon the task
    used) by running process(s) and thread(s) where task runs inside thread(s)
    and thread(s) runs inside process(s).

    Args:
      task: Implementation of task.LoadTestTask.

    Returns:
      Returns start_time, end_time of load test, latencies and results of all
        the tasks performed over the span of load test.

    Raises:
      RuntimeError: If the given task is not completed even once during the
        course of load test.
    """
    tasks_results_queue = multiprocessing.Manager().Queue()
    pre_tasks_results_queue = multiprocessing.Manager().Queue()
    post_tasks_results_queue = multiprocessing.Manager().Queue()

    processes = []
    for process_id in range(self.num_processes):
      process = multiprocessing.Process(
          target=LoadGenerator._process_task,
          args=(task, process_id, self.num_threads_per_process,
                self.num_executions_per_thread, pre_tasks_results_queue,
                tasks_results_queue, post_tasks_results_queue))
      processes.append(process)

    # Initialize checkpoints to show completion of load test i.e. 25%, 50% etc.
    # Note: Completion checkpoints are shown only when self.run_time is set.
    # E.g. if self.run_time is 60 then the load test will inform that 50% of
    # load test is completed after 30 seconds.
    log_loading = self.run_time != sys.maxsize
    loading_checkpoints = list(
        map(lambda t: (t * self.run_time), lg_const.TIME_LOADING_PERCENTAGES))
    curr_loading_idx = 0

    for process in processes:
      process.start()
    logging.debug('%s number of processes started for task %s', len(processes),
                  task.task_name)

    start_time = curr_time = time.time()
    loading_checkpoints = [t + start_time for t in loading_checkpoints]
    # Loop till the condition of termination of load test is not met. The
    # condition is either the load test has run for self.run_time or completed
    # the total number of tasks assigned.
    while ((curr_time - start_time) < self.run_time) & \
        (tasks_results_queue.qsize() < self.total_num_tasks):
      # Sleep so that the looping is not very fast. 0.1 is decided on
      # discretion with the intention that time duration shouldn't be very
      # small or shouldn't be very large.
      time.sleep(0.1)
      curr_time = time.time()
      if log_loading & (curr_loading_idx < len(loading_checkpoints)) and (
          curr_time >= loading_checkpoints[curr_loading_idx]):
        logging.info('Load test completed %s%% for task: %s',
                     lg_const.TIME_LOADING_PERCENTAGES[curr_loading_idx] * 100,
                     task.task_name)
        curr_loading_idx = curr_loading_idx + 1
    logging.info('Load test completed 100%% for task: %s', task.task_name)

    for process in processes:
      process.terminate()
    logging.debug('%s number of processes terminated for task %s',
                  len(processes), task.task_name)

    # Raise error if not even a single task is completed
    if tasks_results_queue.qsize() < 1:
      raise RuntimeError('Not even a single task is completed. Pass higher '
                         'value to --run-time flag or check the task.')
    return {
        lg_const.START_TIME:
            start_time,
        lg_const.END_TIME:
            curr_time,
        lg_const.TASKS_RESULTS:
            self._convert_multiprocessing_queue_to_list(tasks_results_queue),
        lg_const.PRE_TASKS_RESULTS:
            self._convert_multiprocessing_queue_to_list(pre_tasks_results_queue
                                                       ),
        lg_const.POST_TASKS_RESULTS:
            self._convert_multiprocessing_queue_to_list(post_tasks_results_queue
                                                       ),
    }

  def post_load_generation(self,
                           observations,
                           output_file=None,
                           print_metrics=True):
    """Task to perform after load testing.

    In this default implementation, latency metrics are computed. It can also
    dump and print the metrics.

    Args:
       observations: Observations collected during load generation.
       output_file: String path where the metrics are dumped in JSON.
       print_metrics: Bool, whether to print the metrics on console or not.

    Returns:
      Default metrics (latencies of tasks) in a dictionary.
    """
    metrics = self._compute_default_post_test_metrics(observations)

    # Dump metrics.
    if output_file:
      self._dump_metrics_into_json(metrics, output_file)

    # Print metrics on console
    if print_metrics:
      self._print_default_metrics(metrics)

    return metrics

  @staticmethod
  def _thread_task(task, process_id, thread_id, num_executions_per_thread,
                   pre_tasks_results_queue, tasks_results_queue,
                   post_tasks_results_queue):
    """Task run in threads spawned during the load test.

    The task used for the load test is run inside this thread. Pre- and
    post-task are also run inside this thread.
    This method is kept as protected as it is not used in other classes and
    static because class methods can't be passed as target of thread.
    """
    cnt = 0
    tasks = [task.pre_task, task.task, task.post_task]
    queues = [
        pre_tasks_results_queue, tasks_results_queue, post_tasks_results_queue
    ]
    while cnt < num_executions_per_thread:
      for curr_task, curr_queue in zip(tasks, queues):
        start_time = time.time()
        result = curr_task(process_id, thread_id)
        end_time = time.time()
        curr_queue.put(
            TaskExecutionResult(
                process_id=process_id,
                thread_id=thread_id,
                start_time=start_time,
                end_time=end_time,
                result=result))
      cnt = cnt + 1

  @staticmethod
  def _process_task(task, process_id, num_threads_per_process,
                    num_executions_per_thread, pre_tasks_results_queue,
                    tasks_results_queue, post_tasks_results_queue):
    """Task run in processes spawned during the load test.

    It spawns num_threads_per_process number of threads where each thread runs
    the task of load test.
    This method is kept as protected as it is not used in other classes and
    static because class methods can't be passed as target of process.
    """
    # Spawn threads that will run task of load test.
    threads = []
    for thread_num in range(num_threads_per_process):
      threads.append(
          threading.Thread(
              target=LoadGenerator._thread_task,
              args=(task, process_id, thread_num, num_executions_per_thread,
                    pre_tasks_results_queue, tasks_results_queue,
                    post_tasks_results_queue)))

    for thread in threads:
      # Thread is kept as daemon, so that it is killed when the parent process
      # is killed.
      thread.daemon = True
      thread.start()
    logging.debug('Threads started for process number: %s', process_id)

    for thread in threads:
      thread.join()
    logging.debug('Threads tasks completed for process number: %s', process_id)

  def _convert_multiprocessing_queue_to_list(self, mp_queue):
    """Converts the multiprocessing queue to list.
    """
    queue_size = mp_queue.qsize()
    return [mp_queue.get() for _ in range(queue_size)]

  def _compute_percentiles(self, data_pts):
    """Compute percentiles for given data points.

    Args:
      data_pts: List of integer data points.

    Returns:
      Dictionary containing 25, 50, 90, 95 & 99 percentiles along with min,
      max and mean.
    """
    np_array = np.array(data_pts)
    return {
        lg_const.MIN: min(data_pts),
        lg_const.MEAN: np.mean(np_array),
        lg_const.MAX: max(data_pts),
        lg_const.PER_25: np.percentile(np_array, 25),
        lg_const.PER_50: np.percentile(np_array, 50),
        lg_const.PER_90: np.percentile(np_array, 90),
        lg_const.PER_95: np.percentile(np_array, 95),
        lg_const.PER_99: np.percentile(np_array, 99)
    }

  def _compute_default_post_test_metrics(self, observations):
    """Computes default post load test metrics using observations.

    Computes latency related metrics (percentiles, min, max and mean) of tasks
    performed over the course of load test.
    """
    # Time stamps
    start_time = observations[lg_const.START_TIME]
    end_time = observations[lg_const.END_TIME]
    actual_run_time = end_time - start_time

    # Latency stats
    latency_stats_names = [
        lg_const.PRE_TASKS_LAT_STATS, lg_const.TASKS_LAT_STATS,
        lg_const.POST_TASKS_LAT_STATS
    ]
    result_names = [
        lg_const.PRE_TASKS_RESULTS, lg_const.TASKS_RESULTS,
        lg_const.POST_TASKS_RESULTS
    ]
    latency_stats = {}
    for stat_name, result_name in zip(latency_stats_names, result_names):
      lat_pts = [
          result.end_time - result.start_time
          for result in observations[result_name]
      ]
      lat_pers = self._compute_percentiles(lat_pts)
      latency_stats[stat_name] = lat_pers

    metrics = {
        lg_const.START_TIME: start_time,
        lg_const.END_TIME: end_time,
        lg_const.ACTUAL_RUN_TIME: actual_run_time,
        lg_const.TASKS_COUNT: len(observations[lg_const.TASKS_RESULTS])
    }
    metrics.update(latency_stats)
    return metrics

  def _dump_metrics_into_json(self, metrics, output_file):
    """Dumps given metrics as JSON file in UTF-8 format given output directory.
    """
    with open(output_file, 'w', encoding='utf-8') as f_p:
      json.dump(metrics, f_p)

  def _print_default_metrics(self, metrics):
    """Prints given default metrics to console.
    """
    actual_run_time = metrics[lg_const.ACTUAL_RUN_TIME]
    latency_stats_names = [
        lg_const.PRE_TASKS_LAT_STATS, lg_const.TASKS_LAT_STATS,
        lg_const.POST_TASKS_LAT_STATS
    ]
    task_type_names = ['Pre', 'Task', 'Post']

    # Time metrics
    print('\nTime: ')
    print('\tStart time (epoch): ', metrics[lg_const.START_TIME])
    print('\tEnd time (epoch): ', metrics[lg_const.END_TIME])
    print('\tActual run time (in seconds): ', actual_run_time)

    # Task related
    print('\nTasks: ')
    print('\tTasks count: ', metrics[lg_const.TASKS_COUNT])
    print('\tTasks per sec: ', metrics[lg_const.TASKS_COUNT] / actual_run_time)

    # Latency metrics
    print('\nTasks latencies: ')
    for task_type, latency_stat_name in zip(task_type_names,
                                            latency_stats_names):
      print('\t', task_type, ': ')
      print('\t\tMin (in seconds): ', metrics[latency_stat_name][lg_const.MIN])
      print('\t\tMean (in seconds): ', metrics[latency_stat_name][lg_const.MAX])
      print('\t\t25th Percentile (in seconds): ',
            metrics[latency_stat_name][lg_const.PER_25])
      print('\t\t50th Percentile (in seconds): ',
            metrics[latency_stat_name][lg_const.PER_50])
      print('\t\t90th Percentile (in seconds): ',
            metrics[latency_stat_name][lg_const.PER_90])
      print('\t\t95th Percentile (in seconds): ',
            metrics[latency_stat_name][lg_const.PER_95])
      print('\t\t99th Percentile (in seconds): ',
            metrics[latency_stat_name][lg_const.PER_99])
      print('\t\tMax (in seconds): ', metrics[latency_stat_name][lg_const.MAX])
