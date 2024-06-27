import csv
import statistics as stats
import numpy as np


def calls_data_writer(obj, file, call_type):
    data = []
    for i in range(len(obj)):
        call_data = []
        call_data.append(obj[i].call_name)
        call_data.append(call_type)
        call_data.append(obj[i].calls_made)
        call_data.append(obj[i].calls_returned)
        call_data.append(obj[i].total_response_time)
        if len(obj[i].response_times) == 0:
            call_data.append(0)
            call_data.append(0)
            call_data.append(0)
            call_data.append(0)
        else:
            call_data.append(int(stats.mean(obj[i].response_times)))
            call_data.append(int(stats.median(obj[i].response_times)))
            call_data.append(int(np.percentile(obj[i].response_times, 90)))
            call_data.append(int(max(obj[i].response_times)))
        data.append(call_data)
    with open(file, 'a', newline='') as f:
        writer = csv.writer(f)
        writer.writerows(data)


def handle_data_writer(global_data, file):
    data = []
    for handle in global_data.handle_name_map.keys():
        name = global_data.handle_name_map[handle]
        obj = global_data.name_object_map[name].handles[handle]
        row = [name, handle, obj.total_reads, obj.total_writes]
        if obj.total_reads != 0:
            row.append(float(obj.total_read_size/obj.total_reads))
            row.append(stats.mean(obj.read_times))
        else:
            row.append(0)
            row.append(0)
        if obj.total_writes != 0:
            row.append(float(obj.total_write_size/obj.total_writes))
            row.append(stats.mean(obj.write_times))
        else:
            row.append(0)
            row.append(0)
        row.append((obj.closing_time - obj.opening_time) + 1e-9*(obj.closing_time_nano - obj.opening_time_nano))
        row.append(obj.opening_time + 1e-9*obj.opening_time_nano)
        row.append(obj.closing_time + 1e-9*obj.closing_time_nano)
        row.append(obj.open_pos)
        row.append(obj.close_pos)
        pattern = []
        type_map = {"r": "random", "s": "sequential"}
        if len(obj.read_pattern) > 1:
            last_read = obj.read_pattern[1]
            streak = 1
            for i in range(2, len(obj.read_pattern)):
                if obj.read_pattern[i] != last_read:
                    read_tup = {}
                    read_tup["read_type"] = type_map[last_read]
                    read_tup["number"] = streak
                    # pattern.append({"read_type": type_map[last_read], "number": streak})
                    pattern.append(read_tup)
                    last_read = obj.read_pattern[i]
                    streak = 1
                else:
                    streak += 1

            # pattern.append({"read_type": type_map[last_read], "number": streak})
            read_tup = {}
            read_tup["read_type"] = type_map[last_read]
            read_tup["number"] = streak
            pattern.append(read_tup)
        row.append(pattern)
        data.append(row)
    with open(file, 'a', newline='') as f:
        writer = csv.writer(f)
        writer.writerows(data)


def main_csv_generator(global_data):
    # global_csv_file = "/usr/local/google/home/patelvishvesh/mount-folder/global_data.csv"
    csv_location = input("Enter the location where you want to save csv: ")
    global_csv_file = csv_location + "global_data.csv"
    data = [['call_name', 'call_type', 'calls_sent', 'calls_responded', 'total_response_time', 'average_response_time', 'median_response_time', 'p90_response_time', 'max_response_time']]
    with open(global_csv_file, 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerows(data)
    calls_data_writer(global_data.gcalls.gcs_calls, global_csv_file, "GCS")
    calls_data_writer(global_data.gcalls.kernel_calls, global_csv_file, "kernel")
    handle_data_csv = csv_location + "handle_data.csv"
    data.clear()
    data = [['file_name', 'handle', 'total_reads', 'total_writes', 'average_read_size', 'average_read_response_time', 'average_write_size', 'average_write_response_time', 'total_request_time', 'opening_time', 'closing_time', 'opened_handles', 'closed_handles', 'read_pattern']]
    with open(handle_data_csv, 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerows(data)
    handle_data_writer(global_data, handle_data_csv)