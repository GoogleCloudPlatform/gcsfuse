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
"""Contains abstract class to represent task in load test.

Example:

    lg_obj = lg.LoadGenerator(...)
    task_obj = TaskImplementingLoadTestTask(...)
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

  def __init__(self):
    # A name given to task. It is recommended to keep a unique name for every
    # task implemented in same module.
    self.task_name = self.__class__.__name__

  def pre_task(self, process_id, thread_id):
    """Task to be always performed before the actual task in a load test.

    process_id & thread_id are kept as arguments in case the task logic
    requires them.

    Args:
      process_id: The process number assigned by the load generator to
        the process running this task.
      thread_id: The process number assigned by the load generator to
        the thread running this task.

    Returns:
      Result of the task. Type can be anything depending upon the use-case.
    """
    pass

  @abstractmethod
  def task(self, process_id, thread_id):
    """Actual task in a load test.

    process_id & thread_id are kept as arguments in case the task logic
    requires them.

    Args:
      process_id: The process number assigned by the load generator to
        the process running this task.
      thread_id: The process number assigned by the load generator to
        the thread running this task.

    Returns:
      Result of the task. Type can be anything depending upon the use-case.
    """
    pass

  def post_task(self, process_id, thread_id):
    """Task to be always performed after the actual task in a load test.

    process_id & thread_id are kept as arguments in case the task logic
    requires them.

    Args:
      process_id: The process number assigned by the load generator to
        the process running this task.
      thread_id: The process number assigned by the load generator to
        the thread running this task.

    Returns:
      Result of the task. Type can be anything depending upon the use-case.
    """
    pass
