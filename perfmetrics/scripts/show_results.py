import json

# Opening JSON file
f = open('output.json')

# returns JSON object as 
# a dictionary
data = json.load(f)

# Iterating through the json
# list

for d in data['jobs']:
  print("filesize: "+ d["job options"]["filesize"])
  if d["read"]["bw"] != 0 :
    if d["job options"]["rw"] == "read":
      print("Read bw: " + str(float(d["read"]["bw"]/1000.0)) + "MiB/s")
    else:
      print("Random read bw: " + str(float(d["read"]["bw"]/1000.0)) + "MiB/s")

  if d["write"]["bw"] != 0 :
    if d["job options"]["rw"] == "write":
      print("Write bw: " + str(float(d["write"]["bw"]/1000.0)) + "MiB/s")
    else:
      print("Random write bw: " + str(float(d["write"]["bw"]/1000.0)) + "MiB/s")
# Closing file
f.close()