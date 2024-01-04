# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Originally written for read cache logs parsing.
This script takes gcsfuse logs in json format and parse them into the following
format:
 {
    "2": {
        "handle": "2",
        "start_time": 1704345315,
        "process_id": "153607",
        "inode_id": "3",
        "object_name": "bucket_name/object_name",
        "chunks": [
            {
                "start_time": 1704345315,
                "start_offset": "0",
                "size": "1048576",
                "cache_hit": "false,",
                "is_sequential": "true,"
            },
            ...
        ]
    },
...
}
"""

import sys
import json
import re


def read_file_line_by_line(filename):
  """
  Reads a file line by line and returns it.
  Args:
      filename (str): The path to the file to be read.
  Yields:
      str: Each line of the file.
  """
  with open(filename, 'r') as file:
    for line in file:
      yield line.strip()  # Yield the line without trailing newline
  file.close()


def filter_and_parse_log_line(log_line, dictionary):
  """
  Filters out Filecache logs and parses them.
  Args:
      log_line (str): single json format logs written by GCSFuse.
      dictionary (dict): stores the structured logs.
  Yields:
      nil
  """
  json_log = json.loads(log_line)
  filter = ["FileCache OK"]

  # Filters out read cache logs.
  if any(x in json_log["msg"] for x in filter):
    # Remove any redundant spaces from the logs.
    json_log["msg"] = re.sub("\s\s+", " ", json_log["msg"])
    # Splitting the logs on whitespace.
    tokenized_logs = json_log["msg"].split(" ")
    # Parse tokenized logs to get the structured logs.
    parse_cache_log(json_log, tokenized_logs, dictionary)


def parse_cache_log(json_log, tokenized_logs, dictionary):
  """
  Parses tokenized GCSFuse logs to get the required structure.
  Args:
      json_log (dict): single json format logs written by GCSFuse.
      tokenized_logs (list): list containing tokens from GCSFuse log message.
      dictionary (dict): stores the structured logs.
  Yields:
      nil
  """
  start_timestamp = json_log['time']['timestampNanos']
  is_sequential = tokenized_logs[5][:-1]  # Remove trailing ","
  cache_hit = tokenized_logs[7][:-1]  # Remove trailing ","
  pid = tokenized_logs[9][:-1]  # Remove trailing ","
  inode = tokenized_logs[11][:-1]  # Remove trailing ","
  handle = tokenized_logs[13][:-1]  # Remove trailing ","
  offset = tokenized_logs[15][:-1]  # Remove trailing ","
  size = tokenized_logs[17][:-1]  # Remove trailing ","
  object_name = tokenized_logs[19][:- 1]  # Remove trailing ")"

  chunk_data = {
      "start_time": start_timestamp,
      "start_offset": offset,
      "size": size,
      "cache_hit": cache_hit,
      "is_sequential": is_sequential,
  }

  dictionary.setdefault(handle, {
      "handle": handle,
      "start_time": start_timestamp,
      "process_id": pid,
      "inode_id": inode,
      "object_name": object_name,
      "chunks": []
  })["chunks"].append(chunk_data)


def main():
  """Main function of the script."""

  # Retrieve the arguments passed to the script
  log_file_path = str(sys.argv[1])
  output_file_path = str(sys.argv[2])

  # dictionary stores the structured logs.
  dictionary = dict()

  # Parse the log file provided in argument
  for line in read_file_line_by_line(log_file_path):
    filter_and_parse_log_line(line, dictionary)

  # Serializing dictionary into json
  json_object = json.dumps(dictionary, indent=4)
  # Writing the structured json logs to output file.
  with open(output_file_path, "w") as outfile:
    outfile.write(json_object)


if __name__ == "__main__":
  main()
