import zipfile
import json
import csv
import datetime
import gzip
import os


class GetLogs:
    def iso_to_epoch(self, timestamp_str):
        """
        converts iso to epoch time (seconds and nanoseconds)
        :param timestamp_str: string with iso time
        :return: epoch time json object
        """
        try:
            datetime_obj = datetime.datetime.fromisoformat(timestamp_str)
            seconds = int(datetime_obj.timestamp())
            nanos = datetime_obj.microsecond * 1000
            return {"seconds": seconds, "nanos": nanos}
        except ValueError as e:
            print(f"Error parsing timestamp: {e}")
            return None

    def get_sorted_files(self, files, log_type, log_format):
        """
        for each file it reads the first log with valid timestamp and then arranges
        the files in order of timestamps
        :param files: list of file names
        :param log_type: gke/gcsfuse
        :param log_format: json/csv
        :return: ordered list of files
        """
        unordered_list = []
        for file in files[:]:
            if file.find(".zip") != -1:
                destination_dir = file[0:file.rfind("/")+1]
                zip_ref = zipfile.ZipFile(file, "r")
                zip_list = zip_ref.namelist()
                zip_ref.extractall(destination_dir)
                os.remove(file)
                # adding the extracted files to the files
                for zipf in zip_list:
                    files.append(destination_dir+zipf)
            elif file.endswith(".gz"):
                destination_dir = file[:file.rfind("/") + 1]
                with gzip.open(file, 'rb') as f_in:
                    uncompressed_name = os.path.join(destination_dir, os.path.basename(file)[:-3])  # Remove .gz extension
                    with open(uncompressed_name, 'wb') as f_out:
                        f_out.write(f_in.read())
                os.remove(file)  # Delete the original .gz file after extraction
                files.append(uncompressed_name)
            else:
                unordered_list.append(file)
        # to arrange the files sequentially
        file_tuple = []
        pos = 0
        if log_type == "gcsfuse":
            for file in unordered_list:
                if log_format == "JSON":
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
                else:
                    with open(file, "r") as handle:
                        for line in handle:
                            start_ind = line.find("time=\"") + len("time=\"")
                            end_ind = line.find("\"", start_ind)
                            if start_ind != -1 and end_ind != -1:
                                time = line[start_ind:end_ind]
                                date_str, frac_str = time.split('.')
                                datetime_obj = datetime.datetime.strptime(date_str, "%d/%m/%Y %H:%M:%S")
                                epoch_time = datetime_obj.timestamp()
                                precision = len(frac_str)
                                frac = int(frac_str)
                                nanoseconds = int(frac * pow(10, 9 - precision))
                                file_tuple.append([[epoch_time, nanoseconds], pos])
                                break
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

        file_tuple.sort()
        ordered_list = []
        for file_tup in file_tuple:
            ordered_list.append(unordered_list[file_tup[1]])
        return ordered_list

    def append_logs(self, logs, temp_logs, interval):
        """
        extracts timestamp and message (textPayload) and appends to the list of logs
        :param logs: list of logs
        :param temp_logs: logs of a single file with more fields than we need
        """
        first_log = self.iso_to_epoch(temp_logs[0]["timestamp"])
        last_log = self.iso_to_epoch(temp_logs[len(temp_logs) - 1]["timestamp"])
        first_log_time = first_log["seconds"] + 1e-9*first_log["nanos"]
        last_log_time = last_log["seconds"] + 1e-9*last_log["nanos"]
        if first_log_time < last_log_time:
            for obj in temp_logs:
                if "timestamp" in obj.keys() and "textPayload" in obj.keys():
                    time_obj = self.iso_to_epoch(obj["timestamp"])
                    if time_obj["seconds"] < interval[0]:
                        continue
                    elif time_obj["seconds"] > interval[1]:
                        break
                    json_log = {"timestamp": self.iso_to_epoch(obj["timestamp"]), "message": obj["textPayload"]}
                    logs.append(json_log)
        else:
            file_len = len(temp_logs) - 1
            for i in range(len(temp_logs)):
                obj = temp_logs[file_len - i]
                if "timestamp" in obj.keys() and "textPayload" in obj.keys():
                    time_obj = self.iso_to_epoch(obj["timestamp"])
                    if time_obj["seconds"] < interval[0]:
                        continue
                    elif time_obj["seconds"] > interval[1]:
                        break
                    json_log = {"timestamp": self.iso_to_epoch(obj["timestamp"]), "message": obj["textPayload"]}
                    logs.append(json_log)

    def get_json_logs(self, files, log_type, interval, log_format):
        """
        calls get_sorted_files to get sorted files and the depending on the format
        it opens file and calls append_logs or just directly appends logs (for json gcsfuse logs)
        :param files: list of file names
        :param log_type: gke/gcsfuse
        :param interval: time interval for which logs are wanted
        :param log_format: json/csv
        :return: a list of json logs with two fields message and timestamp(epoch)
        """
        ordered_files = self.get_sorted_files(files, log_type, log_format)
        logs = []
        for file in ordered_files:
            if log_type == "gcsfuse":
                if log_format == "JSON":
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
                else:
                    with open(file, 'r') as handle:
                        for line in handle:
                            start_ind = line.find("time=\"") + len("time=\"")
                            end_ind = line.find("\"", start_ind)
                            start_ind1 = line.find("message=\"") + len("message=\"")
                            end_ind1 = line.rfind("\"")
                            if start_ind1 != -1 and end_ind1 != -1 and start_ind != -1 and end_ind != -1:
                                time = line[start_ind:end_ind]
                                message = line[start_ind1:end_ind1]
                                message = message.replace(r"\"", "\"")
                                date_str, frac_str = time.split('.')
                                datetime_obj = datetime.datetime.strptime(date_str, "%d/%m/%Y %H:%M:%S")
                                epoch_time = datetime_obj.timestamp()
                                precision = len(frac_str)
                                frac = int(frac_str)
                                nanoseconds = int(frac * pow(10, 9 - precision))
                                if epoch_time < interval[0]:
                                    continue
                                elif epoch_time > interval[1]:
                                    break
                                log_data = {"timestamp": {"seconds": epoch_time, "nanos": nanoseconds}, "message": message}
                                logs.append(log_data)

            elif log_type == "gke":
                if log_format == "JSON":
                    with open(file, "r") as handle:
                        data = json.load(handle)
                    if not isinstance(data, list):
                        raise ValueError("Expected a JSON list in the file")
                    self.append_logs(logs, data, interval)
                else:
                    temp_logs = []
                    with open(file, 'r') as csvfile:
                        reader = csv.reader(csvfile)
                        header_row = next(reader)
                        fields_to_extract = ["timestamp", "textPayload"]
                        field_indices = {field: header_row.index(field) for field in fields_to_extract if field in header_row}
                        for row in reader:
                            json_log = {"timestamp": row[field_indices["timestamp"]], "textPayload": row[field_indices["textPayload"]]}
                            temp_logs.append(json_log)
                    self.append_logs(logs, temp_logs, interval)

        return logs
