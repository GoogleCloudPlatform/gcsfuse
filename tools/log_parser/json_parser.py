import sys
import json
import re

log_file_path = str(sys.argv[1])
output_file_name = str(sys.argv[2])


# parsed_logs = dict()

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
      # Process each line here
      yield line.strip()  # Yield the line without trailing newline
  file.close()


dictionary = dict()
def parse(log_line):
  data = json.loads(log_line)
  matches = ["FileCache OK"]

  # Filters out read cache logs.
  if any(x in data["msg"] for x in matches):
    # Remove any redundant spaces from the logs.
    data["msg"] = re.sub("\s\s+", " ", data["msg"])
    # Split on spaces.
    split_data = data["msg"].split(" ")
    parse_cache_log(data, split_data)


def parse_cache_log(data, split_msg):
  startTimestamp = data["time"]["timestampSeconds"]
  op_id = split_msg[0]
  is_sequential = split_msg[5]
  cache_hit = split_msg[7]
  handle = split_msg[9][11:]
  inode = split_msg[10][6:]
  offset = split_msg[11][7:]
  pid = split_msg[12][4:len(split_msg[12]) - 1]
  size = split_msg[13][5:len(split_msg[13]) - 2]
  object_name = split_msg[15][:len(split_msg[15]) - 1]
  print("startTimestamp", startTimestamp, "op_id: ", op_id, "is_sequential: ",
        is_sequential, "cache_hit: ", cache_hit, "handle: ", handle, "inode: ",
        inode, "offset: ", offset, "pid: ", pid, "size", size, "object",
        object_name)

  if dictionary.get("handle") is None:
    print("not in dict")
    dictionary["handle"] = handle
    dictionary["start_time"] = startTimestamp
    dictionary["process_id"] = pid
    dictionary["inode_id"] = inode
    dictionary["object_name"] = object_name
    dictionary["chunks"] = {
        "start_time": startTimestamp,
        "start_offset": offset,
        "size": size,
        "cache_hit": cache_hit,
        "is_sequential": is_sequential
    }
    print(dictionary)

for line in read_file_line_by_line(log_file_path):
  parse(line)
