import arranging
import process
import json
# Add the log files here
files = []
itr = ""
print("Enter the logs file names (with absolute path), press -1 when done:")
while itr != "-1":
    itr = input()
    if itr != "-1":
        files.append(itr)

print("Entered the time window for which you want the logs to be analysed")
start_time = int(input("start time(epoch): "))
end_time = int(input("end time(epoch): "))

ordered_files = arranging.arrange(files)

# tot_logs = []
# agg_logs = [[], [], []]
logs = []
for file in files:
    with open(file, "r") as handle:
        for line in handle:
            data = line.strip()
            try:
                json_object = json.loads(data)
                logs.append(json_object)
            except json.JSONDecodeError:
                print(f"Error parsing line: {line}")

# tot_logs = log_agg.seg_log(ordered_files, agg_logs, start_time, end_time, file)

process.gen_processor(logs)