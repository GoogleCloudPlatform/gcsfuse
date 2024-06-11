from read_pattern_metric import get_val

def processor(logs):
  bytes_to_gcs = 0
  bytes_from_gcs = 0
  latency_from = 0
  latency_to = 0
  for log in logs:
    message = log["message"]
    if message.find("WriteFile") != -1:
      # {"timestamp":{"seconds":1717677403,"nanos":218487205},"severity":"TRACE","message":"fuse_debug: Op 0x000005e2        connection.go:420] <- WriteFile (inode 17, PID 0, handle 65, offset 4194304, 40 bytes)"}
      bytes_to_gcs += int(get_val(message, ",", " ", "bck", 1))

    # {"timestamp":{"seconds":1717744535,"nanos":781137928},"severity":"TRACE","message":"gcs: Req             0x29: <- Read(\"a.txt\", [0, 2))"}
    # {"timestamp":{"seconds":1717744535,"nanos":983928864},"severity":"TRACE","message":"gcs: Req             0x29: -> Read(\"a.txt\", [0, 2)) (202.798347ms): OK"}
    elif message.find("Read") != -1 and message.find("gcs: Req") != -1:
      if message.find("->") != -1 and message.find("nil") == -1:
        start = int(get_val(message, "[", ",", "fwd", 0))
        final = int(get_val(message, ",", ")", "bck", 1))
        start_index = message.rfind("(")
        if message.find("ms", start_index) != -1:
          latency_from += float(get_val(message, "(", "ms", "bck", 0))
        else:
          latency_from += 1000*float(get_val(message, "(", "s", "bck", 0))
        bytes_from_gcs += final-start
        # print("[", start, ",", final, ")")


  print("Total bytes transferred to gcs:", bytes_to_gcs)
  print("Total bytes transferred from gcs:", bytes_from_gcs)
  print("Total latency due to transferring from gcs:", latency_from)
