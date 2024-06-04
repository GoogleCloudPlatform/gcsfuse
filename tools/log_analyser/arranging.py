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
            file_handles.append(open_file(file, "r"))

    # to arrange the files sequentially
    file_tuple = []
    pos = 0
    for entry in file_handles:
        for line in entry:
            data = line.strip()
            try:
                json_object = json.loads(data)
                sec = json_object["timestamp"]["seconds"]
                nano = json_object["timestamp"]["nanos"]
                file_tuple.append([[sec, nano], pos])
                break
            except json.JSONDecodeError:
                print(f"Error parsing line: {line}")
        pos += 1

    # file with the earliest entry gets the first position
    file_tuple.sort()
    ordered_list = []
    for file_tup in file_tuple:
        ordered_list.append(unordered_list[file_tup[1]])

    # closing the log files
    for open_f in file_handles:
        if open_f is not None:
            open_f.close()
    return ordered_list
