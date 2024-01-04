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


def parse_cache_log(data, tokenized_logs, dictionary):
  startTimestamp = data["time"]["timestampSeconds"]
  op_id = tokenized_logs[0]
  is_sequential = tokenized_logs[5]
  cache_hit = tokenized_logs[7]
  handle = tokenized_logs[9][11:]
  inode = tokenized_logs[10][6:]
  offset = tokenized_logs[11][7:]
  pid = tokenized_logs[12][4:len(tokenized_logs[12]) - 1]
  size = tokenized_logs[13][5:len(tokenized_logs[13]) - 2]
  object_name = tokenized_logs[15][:len(tokenized_logs[15]) - 1]
  print("startTimestamp", startTimestamp, "op_id: ", op_id, "is_sequential: ",
        is_sequential, "cache_hit: ", cache_hit, "handle: ", handle, "inode: ",
        inode, "offset: ", offset, "pid: ", pid, "size", size, "object",
        object_name)

  if dictionary.get(handle) is None:
    dictionary[handle] = {
        "handle": handle,
        "start_time": startTimestamp,
        "process_id": pid,
        "inode_id": inode,
        "object_name": object_name,
        "chunks": [{
            "start_time": startTimestamp,
            "start_offset": offset,
            "size": size,
            "cache_hit": cache_hit,
            "is_sequential": is_sequential
        }]
    }
  else:
    chunks = dictionary.get(handle)["chunks"]
    chunks.append({"start_time": startTimestamp,
                   "start_offset": offset,
                   "size": size,
                   "cache_hit": cache_hit,
                   "is_sequential": is_sequential}
                  )
    dictionary.get(handle)["chunks"] = chunks


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
