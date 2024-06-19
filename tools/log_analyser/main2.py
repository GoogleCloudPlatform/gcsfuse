import arranging
import process
import json
import output_metrics
import os

files = []
directory_path = input("Enter the path to the directory containing log files")
for filename in os.listdir(directory_path):
    # Construct the full path to the file
    file_path = os.path.join(directory_path, filename)

    # Check if it's a regular file (not a directory or hidden file)
    if os.path.isfile(file_path):
        files.append(file_path)

# itr = ""
# print("Enter the logs file names (with absolute path), press -1 when done:")
# while itr != "-1":
#     itr = input()
#     if itr != "-1":
#         files.append(itr)

print("Entered the time window for which you want the logs to be analysed")
start_time = int(input("start time(epoch): "))
end_time = int(input("end time(epoch): "))

ordered_files = arranging.arrange(files)

# tot_logs = []
# agg_logs = [[], [], []]
logs = []
for file in ordered_files:
    with open(file, "r") as handle:
        for line in handle:
            data = line.strip()
            try:
                json_object = json.loads(data)
                logs.append(json_object)
            except json.JSONDecodeError:
                print(f"Error parsing line: {line}")

# tot_logs = log_agg.seg_log(ordered_files, agg_logs, start_time, end_time, file)

global_data = process.gen_processor(logs)
output_metrics.gen_output(global_data)