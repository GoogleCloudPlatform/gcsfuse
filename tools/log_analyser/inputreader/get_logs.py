import zipfile
import json
import csv
import datetime


class GetLogs:
    def iso_to_epoch(self, timestamp_str):
        try:
            datetime_obj = datetime.datetime.fromisoformat(timestamp_str)
            seconds = int(datetime_obj.timestamp())
            nanos = datetime_obj.microsecond * 1000
            return {"seconds": seconds, "nanos": nanos}
        except ValueError as e:
            print(f"Error parsing timestamp: {e}")
            return None


    def get_sorted_files(self, files, log_type):
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
        # to arrange the files sequentially
        file_tuple = []
        pos = 0
        if log_type == "gcsfuse":
            for file in unordered_list:
                with open(file, "r") as handle:
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
                pos += 1
        elif log_type == "gke":
            for file in unordered_list:
                with open(file, 'r', newline='') as csvfile:
                    reader = csv.reader(csvfile)
                    next(reader)

                    # Assuming 1st column contains timestamp and 2nd column contains message
                    for row in reader:
                        timestamp = self.iso_to_epoch(row[0])
                        if timestamp is not None:
                            file_tuple.append([[timestamp["seconds"], timestamp["nanos"]], pos])
                            break
                pos += 1
        # file with the earliest entry gets the first position
        file_tuple.sort()
        ordered_list = []
        for file_tup in file_tuple:
            ordered_list.append(unordered_list[file_tup[1]])
        return ordered_list


    def get_json_logs(self, files, log_type, interval):
        ordered_files = self.get_sorted_files(files, log_type)
        logs = []
        for file in ordered_files:
            if log_type == "gcsfuse":
                with open(file, "r") as handle:
                    for line in handle:
                        data = line.strip()
                        try:
                            json_object = json.loads(data)
                            if json_object["timestamp"]["seconds"] < interval[0]:
                                continue
                            elif json_object["timestamp"]["seconds"] > interval[1]:
                                break
                            logs.append(json_object)
                        except json.JSONDecodeError:
                            print(f"Error parsing line: {line}")

            elif log_type == "gke":
                with open(file, 'r', newline='') as csvfile:
                    reader = csv.reader(csvfile)
                    next(reader)

                    # Assuming 1st column contains timestamp and 2nd column contains message
                    for row in reader:
                        timestamp = self.iso_to_epoch(row[0])
                        message = row[1]
                        if timestamp["seconds"] < interval[0]:
                            continue
                        elif timestamp["seconds"] > interval[1]:
                            break
                        logs.append({"timestamp": timestamp, "message": message})
        return logs
