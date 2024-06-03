import arranging
import log_agg
import open_handles_metric
import read_pattern_metric
# import metric3_processor

# Taking input files and filters from the user

# Add the log files here
files = []
s = ""
print("Enter the logs file names (with absolute path), press -1 when done:")
while s != "-1":
    s = input()
    if s != "-1":
        files.append(s)
    print("\n")

# files = ["/usr/local/google/home/patelvishvesh/tmp/testzip.zip"]
print("Entered the time window for which you want the logs to be analysed\n")
start_time = int(input("start time(epoch): "))
end_time = int(input("end time(epoch): "))

file = ""
file = input("Enter the file name for metrics: ")

ordered_files = arranging.arrange(files)
# for file in ordered_files:
#     print(file, "\n")

# metric 1 is open file handles and metric 2 is read patterns
tot_logs = []
agg_logs = [[], [], []]

inodeFilenameMap = log_agg.seg_log(ordered_files, agg_logs, start_time, end_time, file)

open_handles_metric.processor(file, agg_logs[0])

read_pattern_metric.processor(file, agg_logs[1])

# for log in agg_logs[0]:
#     print("ts: ", log["timestamp"]["seconds"], "\n")

# for entry in inodeFilenameMap:
# print("inode: ", entry, " filename: ", inodeFilenameMap[entry], "\n")

# add the duration for which the analysis is needed
# code goes here

