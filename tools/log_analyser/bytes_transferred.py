from read_pattern_metric import get_val

def processor(logs):
  bytes_to_gcs = 0
  bytes_from_gcs = 0
  for log in logs:
    message = log["message"]
    if message.find("WriteFile") != -1:
      # {"timestamp":{"seconds":1717677403,"nanos":218487205},"severity":"TRACE","message":"fuse_debug: Op 0x000005e2        connection.go:420] <- WriteFile (inode 17, PID 0, handle 65, offset 4194304, 40 bytes)"}
      bytes_to_gcs += int(get_val(message, ",", " ", "bck", 1))

  print("Total bytes transferred to gcs:", bytes_to_gcs)
