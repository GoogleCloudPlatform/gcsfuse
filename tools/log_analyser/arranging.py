# program for getting the input

import zipfile
import json
import utility
import csv


def open_file(file, mode):
    try:
        f = open(file, mode)
        return f
    except FileNotFoundError:
        print(f"Error: File not found - {file}")
        return None
    except PermissionError:
        print(f"Error: You don't have permission to access {file}")
        return None
    except Exception as e:
        print(f"An error occurred: {e}")
        return None


def arrange(files, log_type):
    file_handles = []
    unordered_list = []
    for file in files:
        if file.find(".zip") != -1:
            destination_dir = file[0:file.rfind("/")+1]
            zip_ref = zipfile.ZipFile(file, "r")
            zip_list = zip_ref.namelist()
            zip_ref.extractall(destination_dir)
            # adding the extracted files to the files
            for zipf in zip_list:
                files.append(destination_dir+zipf)
        else:
            unordered_list.append(file)
            # print("appended \n", file)
            # file_handles.append(open_file(file, "r"))

    # to arrange the files sequentially
    file_tuple = []
    pos = 0
    if log_type == "gcsfuse":
        for file in unordered_list:
            with open(file, "r") as handle:
                current_message = None
                current_timestamp = None
                for line in handle:
                    data = line.strip()
                    try:
                        json_object = json.loads(data)
                        sec = json_object["timestamp"]["seconds"]
                        nano = json_object["timestamp"]["nanos"]
                        file_tuple.append([[sec, nano], pos])
                        break
                    except json.JSONDecodeError:
                        print(f"Error parsing line: {line}")
                    # line = line.strip()
                    # if line:
                    #     if line.startswith("20"):
                    #         if current_message:
                    #             log_data = {
                    #                 "timestamp": utility.iso_to_epoch(current_timestamp),
                    #                 "message": current_message,
                    #             }
                    #             if log_data["timestamp"] is not None:
                    #                 file_tuple.append([[log_data["timestamp"]["seconds"], log_data["timestamp"]["nanos"]], pos])
                    #                 break
                    #         current_timestamp = line
                    #         current_message = None
                    #     else:
                    #         current_message = line
            pos += 1
    elif log_type == "gke":
        for file in unordered_list:
            with open(file, 'r', newline='') as csvfile:
                reader = csv.reader(csvfile)
                next(reader)

                # Assuming 1st column contains timestamp and 2nd column contains message
                for row in reader:
                    timestamp = utility.iso_to_epoch(row[0])
                    # message = row[1]
                    # if timestamp["seconds"] < start_time:
                    #     continue
                    # elif timestamp["seconds"] > end_time:
                    #     break
                    if timestamp is not None:
                        file_tuple.append([[timestamp["seconds"], timestamp["nanos"]], pos])
                        break
            pos += 1

    # file with the earliest entry gets the first position
    file_tuple.sort()
    ordered_list = []
    for file_tup in file_tuple:
        ordered_list.append(unordered_list[file_tup[1]])

    # # closing the log files
    # for open_f in file_handles:
    #     if open_f is not None:
    #         open_f.close()
    return ordered_list
