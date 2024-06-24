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


def main_csv_generator(global_data):
    # global_csv_file = "/usr/local/google/home/patelvishvesh/mount-folder/global_data.csv"
    global_csv_file = input("Enter the location where you want to save csv: ")
    global_csv_file += "global_data.csv"
    data = [['Name', 'Type', 'Calls_sent', 'Calls_responded', 'Total_response_time', 'Average_response_time', 'Median_response_time', 'p10_response time', 'Max response time']]
    with open(global_csv_file, 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerows(data)
    calls_data_writer(global_data.gcalls.gcs_calls, global_csv_file, "GCS")
    calls_data_writer(global_data.gcalls.kernel_calls, global_csv_file, "kernel")
