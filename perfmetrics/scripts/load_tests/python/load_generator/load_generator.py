"""Load Generator for performing load test with a given task (in python).

Contains the class for generating load with the help of a task written in
python.

Example:

    task_obj = ReadTask()
    lg_obj = LoadGenerator(...)
    lg_obj.pre_load_task(...)
    observations = lg_obj.generate_load(task_obj)
    metrics = lg_obj.post_load_task(observations, ...)
"""
import logging
import os
import sys
import threading
import time
import json
import multiprocessing
import matplotlib.pyplot as plt
import numpy as np
import psutil

MIN_RUN_TIME_IN_SECS = 4
MIN_OBSERVATION_INTERVAL_IN_SECS = 2
KB = 1024
MB = 1024 * 1024
GB = 1024 * 1024


class LoadGenerator:
  """Generates load with the with a given task.

  Generates load on CPU and other resources depending upon the given task. This
  class also provides default implementation of post load test task to get CPU
  and network and bandwidth of test. Classes derived from this class can define
  their own pre- and post-load test tasks and run before and after actual load
  test.

  Args:
    num_processes: An integer defining the number of processes to run during
      load test.
    num_threads: An integer defining the number of threads to run in each
      process during load test.
    run_time: An integer defining the number of seconds to run the load test
      for.
    num_tasks_per_thread: An integer defining the number of times the given
      task to be run inside each thread of each process during load test.
    num_tasks: An integer defining the total number of times the given task
      to be run across all threads of all processes during load test.
    observation_interval: An integer defining the interval in seconds, such that
      the resources' usage are observed after every interval during load test.
  """

  def __init__(self,
               num_processes,
               num_threads,
               run_time=sys.maxsize,
               num_tasks_per_thread=sys.maxsize,
               num_tasks=sys.maxsize,
               observation_interval=MIN_OBSERVATION_INTERVAL_IN_SECS):
    self.num_processes = num_processes
    self.num_threads = num_threads
    self.run_time = min(sys.maxsize, run_time)
    self.num_tasks_per_thread = min(sys.maxsize, num_tasks_per_thread)
    self.num_tasks = min(sys.maxsize, num_tasks)
    self.observation_interval = observation_interval

    if (self.run_time == sys.maxsize) & \
        (self.num_tasks_per_thread == sys.maxsize) & \
        (self.num_tasks == sys.maxsize):
      raise ValueError('Out of run_time, num_tasks_per_thread and num_threads '
                       'arguments, one has to be passed.')

    if (self.num_tasks_per_thread != sys.maxsize) & (self.num_tasks !=
                                                     sys.maxsize):
      raise ValueError(
          "Arguments num_tasks_per_thread and num_tasks both can't be passed.")

    # Run time should be at least MIN_RUN_TIME_IN_SECS in all cases even if
    # num_tasks_per_thread or num_tasks is set.
    if self.run_time < MIN_RUN_TIME_IN_SECS:
      logging.warning(
          'run_time should be at least %s. Overriding it to %s '
          'for this run.', MIN_RUN_TIME_IN_SECS, MIN_RUN_TIME_IN_SECS)
      self.run_time = MIN_RUN_TIME_IN_SECS

    if observation_interval < MIN_OBSERVATION_INTERVAL_IN_SECS:
      raise ValueError("observation_interval can't be less than "
                       f'{MIN_OBSERVATION_INTERVAL_IN_SECS}')

    # observation interval should be at least self.run_time / 2, so that
    # at least two observations are taken.
    max_observation_interval = self.run_time / 2
    if observation_interval >= max_observation_interval:
      raise ValueError("observation_interval can't be more than "
                       f"('run_time' / 2) i.e. {max_observation_interval}")

    if self.num_tasks != sys.maxsize:
      logging.warning('Note: Actual number of tasks may not be equal to '
                      '--num-tasks passed, specially for low value of '
                      '--num-tasks.')

  def pre_load_test(self, **kwargs):
    """Task to perform before running load test.

    Args:
      kwargs: The arguments are kept dynamic, so that any implementation can be
        done in derived class depending upon use case.
    """
    pass

  def generate_load(self, task):
    """Generates load with the given task.

    The load is generated on CPU and other resources (depending upon the task
    used) by running process(s) and thread(s) where task runs inside thread(s)
    and thread(s) runs inside process(s).

    Args:
      task: Implementation of task.LoadTestTask.

    Returns:
      Returns observations of resources' usage collected over the course of
      load test.

    Raises:
      RuntimeError: If the given task is not completed even once during the
        course of load test.
    """
    tasks_results_queue = multiprocessing.Manager().Queue()
    pre_tasks_results_queue = multiprocessing.Manager().Queue()
    post_tasks_results_queue = multiprocessing.Manager().Queue()

    processes = []
    process_pids = []
    for process_id in range(self.num_processes):
      process = multiprocessing.Process(
          target=LoadGenerator._process_task,
          args=(task, process_id, self.num_threads, self.num_tasks_per_thread,
                pre_tasks_results_queue, tasks_results_queue,
                post_tasks_results_queue))
      processes.append(process)
      process_pids.append(process.pid)

    # Ignore the first psutil.cpu_percent call's output.
    psutil.cpu_percent()
    # Base resources' observations.
    cpu_usage_pts = [
        psutil.cpu_percent(interval=self.observation_interval, percpu=True)
    ]
    net_tcp_conns_pts = [psutil.net_connections(kind='tcp')]
    net_io_pts = [psutil.net_io_counters(pernic=True)]

    # checkpoints initialisation. Note: Only used if self.run_time is set.
    log_loading = self.run_time != sys.maxsize
    loading_percentages = [0.25, 0.50, 0.75]
    loading_checkpoints = list(
        map(lambda t: (t * self.run_time), [0.25, 0.50, 0.75]))
    curr_loading_idx = 0

    for process in processes:
      process.start()
    logging.debug('%s number of processes started for task %s', len(processes),
                  task.TASK_NAME)

    # Observe resources' usages over duration of load test
    # time_pts[0] is start time of load generation.
    time_pts = [time.perf_counter()]
    loading_checkpoints = [t + time_pts[0] for t in loading_checkpoints]
    while self._should_continue_test(time_pts[-1], time_pts[0],
                                     tasks_results_queue.qsize(), processes):
      # psutil.cpu_percent is blocking call for the process (and its core) in
      # which it runs. This blocking behavior also acts as a gap between
      # recording observations for other metrics.
      cpu_usage_pts.append(
          psutil.cpu_percent(interval=self.observation_interval, percpu=True))
      net_tcp_conns_pts.append(psutil.net_connections(kind='tcp'))
      net_io_pts.append(psutil.net_io_counters(pernic=True))
      curr_time = time.perf_counter()
      time_pts.append(curr_time)
      if log_loading & (curr_loading_idx < len(loading_checkpoints)) and (
          curr_time >= loading_checkpoints[curr_loading_idx]):
        logging.info('Load test completed %s%% for task: %s',
                     loading_percentages[curr_loading_idx] * 100,
                     task.TASK_NAME)
        curr_loading_idx = curr_loading_idx + 1

    logging.info('Load test completed 100%% for task: %s', task.TASK_NAME)

    for process in processes:
      process.terminate()
    logging.debug('%s number of processes terminated for task %s',
                  len(processes), task.TASK_NAME)

    # Raise error if not even a single task is completed
    if tasks_results_queue.qsize() < 1:
      raise RuntimeError('Not even a single task is completed. Pass higher '
                         'value to --run-time flag or check the task.')
    return {
        'time_pts':
            time_pts,
        'process_pids':
            process_pids,
        'tasks_results':
            self._convert_multiprocessing_queue_to_list(tasks_results_queue),
        'pre_tasks_results':
            self._convert_multiprocessing_queue_to_list(pre_tasks_results_queue
                                                       ),
        'post_tasks_results':
            self._convert_multiprocessing_queue_to_list(post_tasks_results_queue
                                                       ),
        'cpu_usage_pts':
            cpu_usage_pts,
        'net_io_pts':
            net_io_pts,
        'net_tcp_conns_pts':
            net_tcp_conns_pts,
    }

  def post_load_test(self,
                     observations,
                     output_dir='./',
                     dump_metrics=True,
                     print_metrics=True):
    """Task to perform after load testing.

    In this default implementation, common metrics including CPU usage,
    latencies and network bandwidth are computed from observations. It can also
    dump and print the metrics, bandwidth variation and cpu variation charts.

    Args:
       observations: Observations collected during load generation.
       output_dir: String path where the metrics and charts are dumped.
       dump_metrics: Bool, whether to dump the metrics and charts or not.
       print_metrics: Bool, whether to print the metrics on console or not.

    Returns:
      Default metrics (CPU and network bandwidth) in a dictionary.
    """
    metrics = self._compute_default_post_test_metrics(observations)
    # Get matplotlib charts for upload and download.
    bw_fig = self._get_matplotlib_line_chart(
        metrics['time_pts'],
        [metrics['upload_bw_pts'], metrics['download_bw_pts']],
        'Bandwidth (MiB/sec)', 'time (sec)', 'MiB/sec', ['Upload', 'Download'])
    cpu_fig = self._get_matplotlib_line_chart(metrics['time_pts'],
                                              [metrics['cpu_usage_pts']],
                                              'CPU Usage (%)', 'time (sec)',
                                              '%', ['CPU'])

    # Dump metrics and charts.
    if dump_metrics:
      self._dump_metrics_into_json(metrics, output_dir)
      bw_fig.savefig(os.path.join(output_dir, 'net_bandwidth_variation.png'))
      cpu_fig.savefig(os.path.join(output_dir, 'cpu_variation.png'))

    # Print metrics on console
    if print_metrics:
      self._print_default_metrics(metrics)

    return {'metrics': metrics, 'plots': {'bandwidth': bw_fig, 'cpu': cpu_fig}}

  @staticmethod
  def _thread_task(task, assigned_process_id, assigned_thread_id,
                   num_tasks_per_thread, pre_tasks_results_queue,
                   tasks_results_queue, post_tasks_results_queue):
    """Task run in threads spawned during the load test.

    The task used for the load test is run inside this thread. Pre- and
    post-task are also run inside this task.
    This method is kept as protected as it is not used in other classes and
    static because class methods can't be passed as target of thread.
    """
    cnt = 0
    tasks = [task.pre_task, task.task, task.post_task]
    queues = [
        pre_tasks_results_queue, tasks_results_queue, post_tasks_results_queue
    ]
    while cnt < num_tasks_per_thread:
      for curr_task, curr_queue in zip(tasks, queues):
        start_time = time.perf_counter()
        result = curr_task(assigned_thread_id, assigned_process_id)
        end_time = time.perf_counter()
        curr_queue.put((assigned_process_id, assigned_thread_id, start_time,
                        end_time, result))
      cnt = cnt + 1

  @staticmethod
  def _process_task(task, assigned_process_id, num_threads,
                    num_tasks_per_thread, pre_tasks_results_queue,
                    tasks_results_queue, post_tasks_results_queue):
    """Task run in processes spawned during the load test.

    It spawns num_threads number of threads where each thread runs the task of
    load test.
    This method is kept as protected as it is not used in other classes and
    static because class methods can't be passed as target of process.
    """
    # Spawn threads that will run task of load test.
    threads = []
    for thread_num in range(num_threads):
      threads.append(
          threading.Thread(
              target=LoadGenerator._thread_task,
              args=(task, assigned_process_id, thread_num, num_tasks_per_thread,
                    pre_tasks_results_queue, tasks_results_queue,
                    post_tasks_results_queue)))

    for thread in threads:
      # Thread is kept as daemon, so that it is killed when the parent process
      # is killed.
      thread.daemon = True
      thread.start()

    logging.debug('Threads started for process number: %s', assigned_process_id)
    for thread in threads:
      thread.join()
    logging.debug('Threads completed for process number: %s',
                  assigned_process_id)

  def _convert_multiprocessing_queue_to_list(self, mp_queue):
    """Converts the multiprocessing queue to list.
    """
    queue_size = mp_queue.qsize()
    return [mp_queue.get() for _ in range(queue_size)]

  def _should_continue_test(self, curr_time, start_time, qsize, processes):
    """Checks whether the load test should continue or not.

    Args:
      curr_time: Integer denoting the current time epoch.
      start_time: Integer denoting the start time epoch of load test.
      qsize: Integer denoting the current size of tasks queue.
      processes: List of process objects that are started as part of load test.

    Returns:
      Bool representing if load test should continue or not.
    """
    # The resources should be observed at least once during the load test.
    if curr_time == start_time:
      return True
    if curr_time - start_time >= self.run_time:
      return False
    if qsize >= self.num_tasks:
      return False
    if self.num_tasks_per_thread == sys.maxsize:
      return True
    # If num_tasks_per_thread is set then continue till all processes have not
    # completed.
    processes_status = map(lambda p: p.is_alive(), processes)
    return any(processes_status)

  def _compute_percentiles(self, data_pts):
    """Compute percentiles for given data points.

    Args:
      data_pts: List of integer data points.

    Returns:
      Dictionary containing 25, 50, 75, 90 and 95 percentiles along with min,
      max and mean.
    """
    np_array = np.array(data_pts)
    return {
        'min': min(data_pts),
        'mean': np.mean(np_array),
        'max': max(data_pts),
        '25': np.percentile(np_array, 25),
        '50': np.percentile(np_array, 50),
        '75': np.percentile(np_array, 75),
        '90': np.percentile(np_array, 90),
        '95': np.percentile(np_array, 95),
        '99': np.percentile(np_array, 99)
    }

  def _get_matplotlib_line_chart(self, x_pts, y_data, title, x_title, y_title,
                                 y_labels):
    """Gives matplotlib line chart for given data and chart metadata.

    Args:
      x_pts: Horizontal axis points.
      y_data: List of vertical axis data for multiple lines.
      title: Chart title.
      x_title: Title for horizontal axis.
      y_title: Title for vertical axis.
      y_labels: list of string names of lines in chart.

    Returns:
      matplotlib figure for requested chart.
    """
    fig, axes = plt.subplots()
    for y_pts, y_label in zip(y_data, y_labels):
      axes.plot(x_pts, y_pts, label=y_label)
    axes.legend(loc='upper right')
    axes.set_xlabel(x_title)
    axes.set_ylabel(y_title)
    axes.set_title(title)
    return fig

  def _compute_default_post_test_metrics(self, observations):
    """Computes default post load test metrics using observations.

    Computes CPU and network related metrics using the observations over the
    load test. Metrics includes CPU usage, latencies of tasks and upload and
    download bandwidths.
    """
    # Time stamps
    start_time = observations['time_pts'][0]
    end_time = observations['time_pts'][-1]
    time_pts = [time_pt - start_time for time_pt in observations['time_pts']]
    actual_run_time = end_time - start_time

    # cpu stats
    cpu_usage_pts = list(map(np.mean, observations['cpu_usage_pts']))
    cpu_usage_pers = self._compute_percentiles(cpu_usage_pts)
    avg_cpu_usage = np.mean(cpu_usage_pts)

    # Network bandwidth stats
    upload_bytes_pts = []
    download_bytes_pts = []
    for net_io_pt in observations['net_io_pts']:
      upload_bytes = 0
      download_bytes = 0
      for _, net_io in net_io_pt.items():
        upload_bytes = upload_bytes + net_io.bytes_sent
        download_bytes = download_bytes + net_io.bytes_recv
      upload_bytes_pts.append(upload_bytes / MB)
      download_bytes_pts.append(download_bytes / MB)
    # Network bandwidth points (MiB/sec)
    upload_bw_pts = [0] + [(upload_bytes_pts[idx + 1] - upload_bytes_pts[idx]) /
                           (time_pts[idx + 1] - time_pts[idx])
                           for idx in range(len(time_pts) - 1)]
    download_bw_pts = [0] + [
        (download_bytes_pts[idx + 1] - download_bytes_pts[idx]) /
        (time_pts[idx + 1] - time_pts[idx]) for idx in range(len(time_pts) - 1)
    ]
    # Avg. Network bandwidth (MiB/sec)
    avg_upload_bw = (upload_bytes_pts[-1] -
                     upload_bytes_pts[0]) / actual_run_time
    avg_download_bw = (download_bytes_pts[-1] -
                       download_bytes_pts[0]) / actual_run_time

    # task latency stats
    task_lat_pts = [
        result[3] - result[2] for result in observations['tasks_results']
    ]
    task_lat_pers = self._compute_percentiles(task_lat_pts)

    return {
        'start_time': start_time,
        'end_time': end_time,
        'tasks_count': len(task_lat_pts),
        'tasks_per_sec': len(task_lat_pts) / actual_run_time,
        'actual_run_time': actual_run_time,
        'time_pts': time_pts,
        'cpu_usage_pts': cpu_usage_pts,
        'cpu_usage_pers': cpu_usage_pers,
        'avg_cpu_usage': avg_cpu_usage,
        'upload_bw_pts': upload_bw_pts,
        'download_bw_pts': download_bw_pts,
        'avg_upload_bw': avg_upload_bw,
        'avg_download_bw': avg_download_bw,
        'task_lat_pers': task_lat_pers
    }

  def _dump_metrics_into_json(self, metrics, output_dir):
    """Dumps given metrics as JSON file in UTF-8 format given output directory.
    """
    with open(
        os.path.join(output_dir, 'metrics.json'), 'w', encoding='utf-8') as f_p:
      json.dump(metrics, f_p)

  def _print_default_metrics(self, metrics):
    """Prints given default metrics to console.
    """
    # Time metrics
    print('\nTime: ')
    print('\tStart time (epoch): ', metrics['start_time'])
    print('\tEnd time (epoch): ', metrics['end_time'])
    print('\tActual run time (in seconds): ', metrics['actual_run_time'])

    # Task related
    print('\nTasks: ')
    print('\tTasks count: ', metrics['tasks_count'])
    print('\tTasks per sec: ', metrics['tasks_per_sec'])

    # Latency metrics
    print('\nTasks latencies: ')
    print('\tMin (in seconds): ', metrics['task_lat_pers']['min'])
    print('\tMean (in seconds): ', metrics['task_lat_pers']['mean'])
    print('\t25th Percentile (in seconds): ', metrics['task_lat_pers']['25'])
    print('\t50th Percentile (in seconds): ', metrics['task_lat_pers']['50'])
    print('\t90th Percentile (in seconds): ', metrics['task_lat_pers']['90'])
    print('\t95th Percentile (in seconds): ', metrics['task_lat_pers']['95'])
    print('\tMax (in seconds): ', metrics['task_lat_pers']['max'])

    # CPU metrics
    print('\nCPU: ')
    print('\tAvg. CPU usage (%): ', metrics['avg_cpu_usage'])
    print('\tPeak CPU usage (%): ', metrics['cpu_usage_pers']['max'])

    # Bandwidth metrics
    print('\nNetwork bandwidth (psutil): ')
    print('\tAvg. Upload Bandwidth (MiB/sec): ', metrics['avg_upload_bw'])
    print('\tAvg. Download Bandwidth (MiB/sec): ', metrics['avg_download_bw'])
