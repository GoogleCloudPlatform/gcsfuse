import matplotlib.pyplot as plt
import math
from matplotlib.ticker import MaxNLocator


def parseit(log):
    # {"timestamp":{"seconds":1716966846,"nanos":92208652},"severity":"TRACE","message":"fuse_debug: Op 0x00004106        connection.go:420] <- OpenFile (inode 10, PID 541742)"}
    message = log["message"]
    start_index = message.find("inode")+6
    end_index = message.find(",", start_index)
    inode = int(message[start_index:end_index])
    start_index = message.find("PID")+4
    end_index = message.find(")", start_index)
    pid = int(message[start_index:end_index])
    return [inode, pid]


def processor(file, logs):
    inode = -1
    last_timestamp = 0
    last_handle_val = 0
    x_axis = []
    y_axis = []
    buff_handles = 0
    rel_time = 1716976290
    labels = []
    req = ""
    for log in logs:
        message = log["message"]
        if message.find("OpenFile") != -1:
            data = parseit(log)
            timestamp_sec = log["timestamp"]["seconds"]
            timestamp_nano = log["timestamp"]["nanos"]
            if data[0] == inode:
                if last_timestamp != timestamp_sec:
                    if last_timestamp != 0:
                        x_axis.append(last_timestamp)
                        labels.append(last_timestamp)
                        y_axis.append(buff_handles+last_handle_val)
                    # else:
                    # x_axis.append(last_timestamp)
                    # y_axis.append(buff_handles+last_handle_val)
                    last_timestamp = timestamp_sec
                    last_handle_val = last_handle_val + buff_handles
                    buff_handles = 1
                else:
                    buff_handles += 1
        else:
            if message.find("LookUpInode") != -1 and message.find(file) != -1:
                start_index = message.find("Op 0x")+3
                end_index = start_index+10
                req = message[start_index:end_index]
            if message.find("OK (inode") != -1:
                start_index = message.find("Op 0x")+3
                end_index = start_index+10
                if req == message[start_index:end_index]:
                    start_index = message.find("(inode ")+7
                    end_index = message.rfind(")")
                    inode = int(message[start_index:end_index])

    x_axis.append(last_timestamp)
    labels.append(last_timestamp)
    y_axis.append(last_handle_val + buff_handles)
    last_handle_val += buff_handles
    labels_y = []
    for i in range(last_handle_val):
        labels_y.append(i+1)
    print("Total handles opened: ", last_handle_val, "\n")
    plt.scatter(x_axis, y_axis)
    # plt.xlim(rel_time, last_timestamp)
    plt.xticks(x_axis, labels)
    plt.yticks(y_axis, labels_y)
    # ax = plt.figure().gca()
    # ax.yaxis.set_major_locator(MaxNLocator(integer=True))
    # ax.xaxis.set_major_locator(MaxNLocator(integer=True))
    # plt.axes().xaxis.set_major_locator(MaxNLocator(integer=True))
    for i, (xi, yi) in enumerate(zip(x_axis, y_axis)):
        plt.annotate(f"({xi}, {yi})", (xi, yi), textcoords="offset points", xytext=(0, 10), fontsize=8)
    plt.show()
