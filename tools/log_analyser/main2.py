import arranging
import process
import json
import output_metrics
import os
import utility
import csv
import generate_csv

files = []
# directory_path = input("Enter the path to the directory containing log files: ")
# for filename in os.listdir(directory_path):
#     # Construct the full path to the file
#     file_path = os.path.join(directory_path, filename)
#
#     # Check if it's a regular file (not a directory or hidden file)
#     if os.path.isfile(file_path):
#         files.append(file_path)

itr = ""
print("Enter the logs file names (with absolute path), press -1 when done:")
while itr != "-1":
    itr = input()
    if itr != "-1":
        files.append(itr)

# print("Entered the time window for which you want the logs to be analysed")
# start_time = int(input("start time(epoch): "))
# end_time = int(input("end time(epoch): "))
log_type = input("Enter the type of logs (gcsfuse/gke): ")

ordered_files = arranging.arrange(files, log_type)

logs = []
current_timestamp = None
current_message = None
for file in ordered_files:
    if log_type == "gcsfuse":
        with open(file, "r") as handle:
            for line in handle:
                data = line.strip()
                try:
                    json_object = json.loads(data)
                    # if json_object["timestamp"]["seconds"] < start_time:
                    #     continue
                    # elif json_object["timestamp"]["seconds"] > end_time:
                    #     break
                    logs.append(json_object)
                except json.JSONDecodeError:
                    print(f"Error parsing line: {line}")
        # with open(file, "r") as handle:
        #     for line in handle:
        #         line = line.strip()
        #         if line:
        #             if line.startswith("20"):
        #                 if current_message:
        #                     log_data = {
        #                         "timestamp": utility.iso_to_epoch(current_timestamp),
        #                         "message": current_message,
        #                     }
        #                     if log_data["timestamp"] is not None:
        #                         logs.append(log_data)
        #                 current_timestamp = line
        #                 current_message = None
        #             else:
        #                 current_message = line
        #
        #         # Store the last message (if any)
        #     if current_message:
        #         log_data = {
        #             "timestamp": utility.iso_to_epoch(current_timestamp),
        #             "message": current_message,
        #         }
        #         if log_data["timestamp"] is not None:
        #             logs.append(log_data)
    elif log_type == "gke":
        with open(file, 'r', newline='') as csvfile:
            reader = csv.reader(csvfile)
            next(reader)

            # Assuming 1st column contains timestamp and 2nd column contains message
            for row in reader:
                timestamp = utility.iso_to_epoch(row[0])
                message = row[1]
                # if timestamp["seconds"] < start_time:
                #     continue
                # elif timestamp["seconds"] > end_time:
                #     break
                logs.append({"timestamp": timestamp, "message": message})


global_data = process.gen_processor(logs)
generate_csv.main_csv_generator(global_data)
# output_metrics.gen_output(global_data)
