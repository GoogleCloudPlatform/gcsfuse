import input
import log_agg

# Add the log files here
files = ["/usr/local/google/home/patelvishvesh/tmp/test_json.json"]
# files = ["/usr/local/google/home/patelvishvesh/tmp/testzip.zip"]
ordered_files = input.arrange(files)
for file in ordered_files:
    print(file, "\n")
# metric 1 is open file handles and metric 2 is read patterns
agg_map = [[], []]
file_map = log_agg.seg_log(ordered_files, agg_map)
for log in agg_map[0]:
    print("ts: ", log["timestamp"]["seconds"], "\n")
for key in file_map:
    print("inode: ", key, " filename: ", file_map[key], "\n")

# add the duration for which the analysis is needed
# code goes here

