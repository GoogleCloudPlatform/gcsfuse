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

    def get_sorted_files(self, files, log_type, log_format):
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
                if log_format == "JSON":
                    with open(file, "r") as handle:
                        data = json.load(handle)
                        for obj in data:
                            timestamp = self.iso_to_epoch(obj["timestamp"])
                            if timestamp is not None:
                                file_tuple.append([[timestamp["seconds"], timestamp["nanos"]], pos])
                                break
                else:
                    with open(file, 'r') as csvfile:
                        reader = csv.reader(csvfile)
                        header_row = next(reader)
                        fields_to_extract = ["timestamp", "textPayload"]
                        field_indices = {field: header_row.index(field) for field in fields_to_extract if field in header_row}

                        for row in reader:
                            timestamp = self.iso_to_epoch(row[field_indices["timestamp"]])
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

    def get_json_logs(self, files, log_type, interval, log_format):
        ordered_files = self.get_sorted_files(files, log_type, log_format)
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
                if log_format == "JSON":
                    with open(file, "r") as handle:
                        data = json.load(handle)
                    if not isinstance(data, list):
                        raise ValueError("Expected a JSON list in the file")
                    for obj in data:
                        if "timestamp" in obj.keys() and "textPayload" in obj.keys():
                            json_log = {"timestamp": self.iso_to_epoch(obj["timestamp"]), "message": obj["textPayload"]}
                            logs.append(json_log)
                else:
                    with open(file, 'r') as csvfile:
                        reader = csv.reader(csvfile)
                        header_row = next(reader)
                        fields_to_extract = ["timestamp", "textPayload"]
                        field_indices = {field: header_row.index(field) for field in fields_to_extract if field in header_row}

                        for row in reader:
                            json_log = {"timestamp": self.iso_to_epoch(row[field_indices["timestamp"]]), "message": row[field_indices["textPayload"]]}
                            logs.append(json_log)
        return logs
