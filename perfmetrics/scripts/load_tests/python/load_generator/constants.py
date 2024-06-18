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
"""Module for constants to be used in load generator module and script.
"""
# Sizes
KB = 1024
MB = 1024 * KB
GB = 1024 * MB

# Used in showing percentage of load test completed (time-wise). e.g 0.25
# represents load test has run 25% of its total run time.
TIME_LOADING_PERCENTAGES = (0.25, 0.50, 0.75)

# Metrics names
START_TIME = 'start_time'
END_TIME = 'end_time'
TASKS_RESULTS = 'tasks_results'
PRE_TASKS_RESULTS = 'pre_tasks_results'
POST_TASKS_RESULTS = 'post_tasks_results'
TASKS_LAT_STATS = 'tasks_lat_stats'
PRE_TASKS_LAT_STATS = 'pre_tasks_lat_stats'
POST_TASKS_LAT_STATS = 'post_tasks_lat_stats'
MIN = 'min'
MAX = 'max'
MEAN = 'mean'
PER_25 = 'per_25'
PER_50 = 'per_50'
PER_90 = 'per_90'
PER_95 = 'per_95'
PER_99 = 'per_99'
TASKS_COUNT = 'tasks_count'
ACTUAL_RUN_TIME = 'actual_run_time'
