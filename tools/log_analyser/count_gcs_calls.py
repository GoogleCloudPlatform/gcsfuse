from read_pattern_metric import get_val

validity_check = [[0, 1, 1, 1, 1, 1, 1], [1, 0, 0, 1, 1, 0, 0]]
calls_name = ["ListObjects", "StatObject", "ComposeObjects", "DeleteObject", "CreateObject", "UpdateObject", "CopyObject"]
latency = [0, 0, 0, 0, 0, 0, 0]
calls_made = [0, 0, 0, 0, 0, 0, 0]
calls_returned = [0, 0, 0, 0, 0, 0, 0]
req_map = [{}, {}, {}, {}, {}, {}, {}]


def not_returned_printer(req_dict, call_type):
    for itr in req_dict.keys():
        print(call_type, " call made at ", req_dict[itr], " did not return in the given interval\n")


def general_call_processor(log, msg_type, file, is_dir):
    message = log["message"]
    log_filename = get_val(message, calls_name[msg_type], "\"", "fwd", 2)
    # print(len(log_filename))
    if log_filename == file and validity_check[is_dir][msg_type] == 1:
        if message.find("<-") != -1:
            req = get_val(message, "0x", " ", "fwd", 0)
            req_map[msg_type][req] = log["timestamp"]["seconds"]
            calls_made[msg_type] += 1
        elif message.find("->") != -1:
            req = get_val(message, "0x", " ", "fwd", 0)
            if req in req_map[msg_type].keys():
                calls_returned[msg_type] += 1
                start_index = message.rfind("(")
                if message.find("ms", start_index) != -1:
                    latency[msg_type] += float(get_val(message, "(", "ms", "bck", 0))
                else:
                    latency[msg_type] += 1000*float(get_val(message, "(", "s", "bck", 0))
                req_map[msg_type].pop(req)


def count_calls(file, logs):

    is_dir = (file[len(file)-1] == '/')
    for log in logs:
        message = log["message"]
        if message.find("ListObjects") != -1:
            general_call_processor(log, 0, file, is_dir)
        elif message.find("StatObject") != -1:
            general_call_processor(log, 1, file, is_dir)
        elif message.find("ComposeObjects") != -1:
            general_call_processor(log, 2, file, is_dir)
        elif message.find("DeleteObject") != -1:
            general_call_processor(log, 3, file, is_dir)
        elif message.find("CreateObject") != -1:
            general_call_processor(log, 4, file, is_dir)
        elif message.find("UpdateObject") != -1:
            general_call_processor(log, 5, file, is_dir)
        elif message.find("CopyObject") != -1:
            general_call_processor(log, 6, file, is_dir)

    for i in range(7):
        if validity_check[is_dir][i] == 1:
            print("Total", calls_name[i], "calls made: ", calls_made[i])
            print("Total", calls_name[i], "calls returned: ", calls_returned[i])
            print("Total latency due to", calls_name[i], "calls: ", latency[i])
            if calls_returned[i] != 0:
                print("Average latency due to", calls_name[i], "calls: ", 1.0*latency[i]/calls_returned[i], "\n")
        not_returned_printer(req_map[i], calls_name[i])
        print("")



