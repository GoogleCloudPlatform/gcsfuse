from read_pattern_metric import get_val


def not_returned_printer(req_dict, call_type):
    for itr in req_dict.keys():
        print(call_type, " call made at ", req_dict[itr], " did not return in the given interval\n")


def count_calls(file, logs):
    latency = [0, 0, 0]
    calls_made = [0, 0, 0]
    calls_returned = [0, 0, 0]
    req_map = [{}, {}, {}]
    is_dir = (file[len(file)-1] == '/')
    for log in logs:
        message = log["message"]
       # {"timestamp":{"seconds":1717664104,"nanos":976222345},"severity":"TRACE","message":"gcs: Req             0x33: <- ListObjects(\"testfile6.txt/\")"}
        if message.find("ListObjects") != -1 and is_dir == True:
            log_dir = get_val(message, "ListObjects", "\"", "fwd", 2)
            # print(len(log_dir))
            if log_dir == file:
                if message.find("<-") != -1:
                    req = get_val(message, "0x", " ", "fwd", 0)
                    req_map[0][req] = log["timestamp"]["seconds"]
                    calls_made[0] += 1
                elif message.find("->") != -1:
                    req = get_val(message, "0x", " ", "fwd", 0)
                    if req in req_map[0].keys():
                        calls_returned[0] += 1
                        latency[0] += float(get_val(message, "(", "ms", "bck", 0))
                        req_map[0].pop(req)
        if message.find("StatObject") != -1 and is_dir == False:
            # {"timestamp":{"seconds":1717664104,"nanos":976249685},"severity":"TRACE","message":"gcs: Req             0x34: <- StatObject(\"testfile6.txt\")"}
            log_file = get_val(message, "StatObject", "\"", "fwd", 2)
            # print(log_file)
            if log_file == file:
                if message.find("<-") != -1:
                    req = get_val(message, "0x", " ", "fwd", 0)
                    req_map[1][req] = log["timestamp"]["seconds"]
                    calls_made[1] += 1
                elif message.find("->") != -1:
                    req = get_val(message, "0x", " ", "fwd", 0)
                    if req in req_map[1].keys():
                        calls_returned[1] += 1
                        latency[1] += float(get_val(message, "(", "ms", "bck", 0))
                        req_map[1].pop(req)

    if is_dir:
        print("Total ListObjects calls made: ", calls_made[0])
        print("Total ListObjects calls returned: ", calls_returned[0])
        print("Total latency due to ListObjects calls: ", latency[0])
        if calls_returned[0] != 0:
            print("Average latency due to ListObjects calls: ", 1.0*latency[0]/calls_returned[0])
        not_returned_printer(req_map[0], "ListObjects")
    if not is_dir:
        print("Total StatObject call made: ", calls_made[1])
        print("Total StatObject call returned: ", calls_returned[1])
        print("Total latency due to StatObject calls: ", latency[1])
        if calls_returned[1] != 0:
            print("Average latency due to StatObject calls: ", 1.0*latency[1]/calls_returned[1])
        not_returned_printer(req_map[1], "StatObject")



