import arranging
import log_agg
import open_handles_metric
import read_pattern_metric

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

file = input("Enter the file name for metrics: ")

ordered_files = arranging.arrange(files)

# metric 1 is open file handles and metric 2 is read patterns
tot_logs = []
agg_logs = [[], [], []]

tot_logs = log_agg.seg_log(ordered_files, agg_logs, start_time, end_time, file)

open_handles_metric.processor(file, agg_logs[0])

read_pattern_metric.processor(file, agg_logs[1])
