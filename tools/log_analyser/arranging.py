# program for getting the input

import zipfile
import json


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


def arrange(files):
    # add the log files you want to open in files
    # print("entered arrange\n")
    file_handles = []
    unordered_list = []
    for file in files:
        if file.find(".zip") != -1:
            zip_ref = zipfile.ZipFile(file, "r")
            zip_list = zip_ref.namelist()
            zip_ref.extractall()
            # adding the extracted files to the files
            for zipf in zip_list:
                files.append(zipf)
        else:
            unordered_list.append(file)
            # print("appended \n", file)
            file_handles.append(open_file(file, "r"))

    # to arrange the files sequentially
    file_tuple = []
    pos = 0
    for entry in file_handles:
        # line = opened.readline()
        lines = json.load(entry)
        try:
            for line in lines:
                sec = line["timestamp"]["seconds"]
                nano = line["timestamp"]["nanos"]
                file_tuple.append([[sec, nano], pos])
                break

        except json.JSONDecodeError as e:
            print(f"Error parsing JSON log line: {e}")
            exit(1)
        pos += 1
    # for tupl in file_tuple:
    #     print("ts: ", tupl[0][0], "index: \n", tupl[1])

    # file with the earliest entry gets the first position
    file_tuple.sort()
    ordered_list = []
    for file_tup in file_tuple:
        ordered_list.append(unordered_list[file_tup[1]])

    # closing the log files
    for open_f in file_handles:
        if open_f is not None:
            open_f.close()
    # print("leaving arranged\n")
    return ordered_list
