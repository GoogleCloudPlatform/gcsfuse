import json
import sys

if __name__ == '__main__':
  argv = sys.argv
  if len(argv) != 2:
    raise TypeError('Incorrect number of arguments.\n'
                    'Usage: '
                    'python3 show_results.py <fio output json filepath>')

  # Opening JSON file
  file = open(argv[1])

  # returns JSON object as
  # a dictionary
  data = json.load(file)

  # Closing file
  file.close()

  # Iterating through the json
  # list
  for d in data['jobs']:
      if d["read"]["bw"] != 0 :
        if d["job options"]["rw"] == "read":
          print("Filesize: "+ d["job options"]["filesize"])
          #Read
          print("Read bw: " + str(d["read"]["bw"]/1024.0) + "MiB/s")
        else:
          #RandomRead
          print("RandomRead bw: " + str(d["read"]["bw"]/1024.0) + "MiB/s")

      if d["write"]["bw"] != 0 :
        if d["job options"]["rw"] == "write":
          #Write
          print("Write bw: " + str(d["write"]["bw"]/1024.0) + "MiB/s")
        else:
          #RandomWrite
          print("RandomWrite bw: " + str(d["write"]["bw"]/1024.0) + "MiB/s")