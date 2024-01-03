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
  matches = ["fuse_debug", "FileCache", "Job"]

  # Filters out read cache logs
  if any(x in data["msg"] for x in matches):
    # Remove any redundant spaces from the logs.
    data["msg"] = re.sub("\s\s+", " ", data["msg"])
    # Split on spaces.
    split_data = data["msg"].split(" ")
    if split_data[0] == "fuse_debug:":
      parse_fuse_debug(dictionary, split_data)

    print(split_data)


def parse_fuse_debug(dictionary, split_data):
  op_id = split_data[2]
  temp_dict = dict()
  if dictionary.get(op_id) != "":
    temp_dict = dictionary.get(op_id)

  if temp_dict.get("op_id") == split_data[2] and temp_dict.get(
      "operation") == "LookUpInode":
    temp_dict["inode"] = split_data[7][:len(split_data[7]) - 1]
    # print(temp_dict)
    dictionary[op_id] = temp_dict
    print(dictionary)
    return
  temp_dict["op_id"] = split_data[2]
  temp_dict["operation"] = split_data[5]
  dictionary[op_id] = temp_dict
  print(dictionary)


print(log_file_path)
for line in read_file_line_by_line(log_file_path):
  parse(line)
