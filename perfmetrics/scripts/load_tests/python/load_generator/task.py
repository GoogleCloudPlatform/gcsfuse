"""Contains abstract class to represent task in load test.

Example:

    lg_obj = lg.LoadGenerator(...)
    task_obj = TaskImplementingLoadTestTask()
    metrics = lg_obj.generate_load(task_obj)
"""
from abc import ABC, abstractmethod


class LoadTestTask(ABC):
  """Abstract class to represent task in load test.

  pre_task represents the task to be performed before performing the actual
  task and post_task represents the task to be performed after the actual task.
  However, pre_task and post_task are not mandatory for a task in a typical load
  test.
  """
  # A name given to task. It is recommended to keep a unique name for every
  # task implemented in same module.
  TASK_NAME = "ABSTRACT_TASK"

  def pre_task(self, assigned_process_id, assigned_thread_id):
    """Task to be always performed before the actual task in a load test.

    assigned_process_id & assigned_thread_id are kept as arguments in case the
    task logic requires them.

    Args:
      assigned_process_id: The process number assigned by the load generator to
        the process running this task.
      assigned_thread_id: The process number assigned by the load generator to
        the thread running this task.

    Returns:
      Result of the task. Type can be anything depending upon the use-case.
    """
    pass

  @abstractmethod
  def task(self, assigned_process_id, assigned_thread_id):
    """Actual task in a load test.

    assigned_process_id & assigned_thread_id are kept as arguments in case the
    task logic requires them.

    Args:
      assigned_process_id: The process number assigned by the load generator to
        the process running this task.
      assigned_thread_id: The process number assigned by the load generator to
        the thread running this task.

    Returns:
      Result of the task. Type can be anything depending upon the use-case.
    """
    pass

  def post_task(self, assigned_process_id, assigned_thread_id):
    """Task to be always performed after the actual task in a load test.

    assigned_process_id & assigned_thread_id are kept as arguments in case the
    task logic requires them.

    Args:
      assigned_process_id: The process number assigned by the load generator to
        the process running this task.
      assigned_thread_id: The process number assigned by the load generator to
        the thread running this task.

    Returns:
      Result of the task. Type can be anything depending upon the use-case.
    """
    pass
