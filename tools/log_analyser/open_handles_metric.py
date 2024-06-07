import matplotlib.pyplot as plt
import datetime
from pytz import timezone
from read_pattern_metric import get_val
import math


# put this function in utility file
def epoch_to_iso(epoch_time):
    datetime_obj = datetime.datetime.fromtimestamp(epoch_time, datetime.timezone.utc)
    ist_tz = timezone('Asia/Kolkata')
    ist_datetime = datetime_obj.astimezone(ist_tz)
    hours = ist_datetime.hour % 24
    minutes = ist_datetime.minute
    seconds = ist_datetime.second
    iso_time = f"{hours:02d}:{minutes:02d}:{seconds:02d}"
    return iso_time


def parseit(log):
    message = log["message"]
    inode = int(get_val(message, "inode", ",", "fwd", 1))
    pid = int(get_val(message, "PID", ")", "fwd", 1))
    return [inode, pid]


def processor(file, logs):
    inode = -1
    last_timestamp = 0
    last_handle_val = 0
    x_axis = []
    y_axis = []
    buff_handles = 0
    labels = []
    req = ""
    req_map = {}
    handle_map = {}
    handles_opened = 0
    handles_closed = 0
    for log in logs:
        message = log["message"]
        if message.find("OpenFile") != -1:
            data = parseit(log)
            timestamp_sec = log["timestamp"]["seconds"]
            if data[0] == inode:
                # if last_timestamp + skip_time <= timestamp_sec
                #     if last_timestamp != 0
                #         x_axis.append(last_timestamp)
                #         labels.append(epoch_to_iso(last_timestamp))
                #         y_axis.append(buff_handles+last_handle_val)
                #         last_timestamp = timestamp_sec
                #         last_handle_val = last_handle_val + buff_handles
                #         buff_handles = 1
                handles_opened += 1
                req = get_val(message, "Op 0x", " ", "fwd", 0)
                req_map[req] = timestamp_sec
                if last_timestamp != timestamp_sec:
                    if last_timestamp != 0:
                        x_axis.append(last_timestamp)
                        labels.append(epoch_to_iso(last_timestamp))
                        y_axis.append(buff_handles+last_handle_val)
                    last_timestamp = timestamp_sec
                    last_handle_val = last_handle_val + buff_handles
                    buff_handles = 1
                else:
                    buff_handles += 1

        elif message.find("OK (Handle") != -1:
            req = get_val(message, "Op 0x", " ", "fwd", 0)
            handle = int(get_val(message, "Handle", ")", "fwd", 1))
            if req in req_map.keys():
                handle_map[handle] = req_map[req]

        elif message.find("ReleaseFileHandle") != -1:
            # {"timestamp":{"seconds":1717577027,"nanos":384694489},"severity":"TRACE","message":"fuse_debug: Op 0x00000012        connection.go:420] <- ReleaseFileHandle (PID 0, Handle 0)"}
            handle = int(get_val(message, ", Handle", ")", "fwd", 1))
            if handle in handle_map.keys():
                timestamp_sec = log["timestamp"]["seconds"]
                handles_closed += 1
                handle_map.pop(handle)
                if last_timestamp != timestamp_sec:
                    last_handle_val += buff_handles
                    x_axis.append(last_timestamp)
                    labels.append(epoch_to_iso(last_timestamp))
                    y_axis.append(last_handle_val)
                    buff_handles = -1
                    last_timestamp = timestamp_sec
                else:
                    buff_handles -= 1

        else:
            if message.find("LookUpInode") != -1 and message.find(file) != -1:
                req = get_val(message, "Op 0x", " ", "fwd", -1)
            if message.find("OK (inode") != -1:
                if req == get_val(message, "Op 0x", " ", "fwd", -1):
                    inode = int(get_val(message, "(inode", ")", "fwd", 1))

    x_axis.append(last_timestamp)
    labels.append(epoch_to_iso(last_timestamp))
    y_axis.append(last_handle_val + buff_handles)
    last_handle_val += buff_handles
    labels_y = []
    for i in y_axis:
        labels_y.append(i)
    print("Total handles opened:", handles_opened)
    print("Total handles closed:", handles_closed)
    for handle in handle_map.keys():
        print("Handle", handle, "opened at time:", epoch_to_iso(handle_map[handle]), "was not released in given time interval")
    plt.scatter(x_axis, y_axis)
    plt.xticks(x_axis, labels)
    plt.yticks(y_axis, labels_y)
    plt.xlabel("time (IST)")
    plt.ylabel("Number of handles")
    for i, (xi, yi) in enumerate(zip(x_axis, y_axis)):
        plt.annotate(f"({xi}, {yi})", (xi, yi), textcoords="offset points", xytext=(0, 10), fontsize=10)
    plt.show()
