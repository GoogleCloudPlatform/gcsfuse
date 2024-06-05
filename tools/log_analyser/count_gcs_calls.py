from read_pattern_metric import get_val


def not_returned_printer(req_dict, call_type):
    for itr in req_dict.keys():
        print(call_type, " call made at ", req_dict[itr], " did not return in the given interval\n")


def count_calls(file, logs):
    latency = [0, 0, 0]
    calls_made = [0, 0, 0]
    calls_returned = [0, 0, 0]
    req_map = [{}, {}, {}]
    for log in logs:
        message = log["message"]
        if message.find(file) != -1:
            if message.find("ListObjects") != -1:
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
            if message.find("StatObject") != -1:
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
    print("Total ListObjects calls made: ", calls_made[0])
    print("Total ListObjects calls returned: ", calls_returned[0])
    print("Total latency due to ListObjects calls: ", latency[0])
    if calls_returned[0] != 0:
        print("Average latency due to ListObjects calls: ", 1.0*latency[0]/calls_returned[0])
    not_returned_printer(req_map[0], "ListObjects")
    print("Total StatObject call made: ", calls_made[1])
    print("Total StatObject call returned: ", calls_returned[1])
    print("Total latency due to StatObject calls: ", latency[1])
    if calls_returned[1] != 0:
        print("Average latency due to StatObject calls: ", 1.0*latency[1]/calls_returned[1])
    not_returned_printer(req_map[1], "StatObject")



