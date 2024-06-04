import matplotlib.pyplot as plt
import datetime
from pytz import timezone
from read_pattern_metric import get_val


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
    for log in logs:
        message = log["message"]
        if message.find("OpenFile") != -1:
            data = parseit(log)
            timestamp_sec = log["timestamp"]["seconds"]
            if data[0] == inode:
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
    print("Total handles opened: ", last_handle_val, "\n")
    plt.scatter(x_axis, y_axis)
    plt.xticks(x_axis, labels)
    plt.yticks(y_axis, labels_y)
    plt.xlabel("time (IST)")
    plt.ylabel("Number of handles")
    for i, (xi, yi) in enumerate(zip(x_axis, y_axis)):
        plt.annotate(f"({xi}, {yi})", (xi, yi), textcoords="offset points", xytext=(0, 10), fontsize=10)
    plt.show()
