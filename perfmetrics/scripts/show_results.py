import json

# Opening JSON file
f = open('output.json')

# returns JSON object as 
# a dictionary
data = json.load(f)

# Iterating through the json
# list

for d in data['jobs']:
  if d["read"]["bw"] != 0 :
    if d["job options"]["rw"] == "read":
      print("Filesize: "+ d["job options"]["filesize"])
      #Read
      print("Read bw: " + str(float(d["read"]["bw"]/1000.0)) + "MiB/s")
    else:
      #RandomRead
      print("RandomRead bw: " + str(float(d["read"]["bw"]/1000.0)) + "MiB/s")

  if d["write"]["bw"] != 0 :
    if d["job options"]["rw"] == "write":
      #Write
      print("Write bw: " + str(float(d["write"]["bw"]/1000.0)) + "MiB/s")
    else:
      #RandomWrite
      print("RandomWrite bw: " + str(float(d["write"]["bw"]/1000.0)) + "MiB/s")
# Closing file
f.close()