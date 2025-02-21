import re
from datetime import datetime

operation_ids = []
response_ids = []
debug = False

def parse_log_line(log_line):
    """Parses a log line and extracts relevant information.

    Args:
        log_line: The log line string.

    Returns:
        A dictionary containing the parsed data, or None if the line 
        doesn't match the expected format.
    """

    pattern = r'time="([^"]+)" severity=(\w+) message="([^"]+)"'
    match = re.search(pattern, log_line)

    if match:
        time_str = match.group(1)
        severity = match.group(2)
        message = match.group(3)

        try:
            # Parse the time string into a datetime object
            dt_object = datetime.strptime(time_str, "%d/%m/%Y %H:%M:%S.%f")

            # Further parse the message (optional, but often useful)
            # message_match = re.search(r'fuse_debug: Op (0x[0-9a-fA-F]+)\s+([^\]]+)\] <- (\w+) \(inode (\d+), PID (\d+)\)', message)
            message_match = re.search(r'fuse_debug: Op (0x[0-9a-fA-F]+)\s+([^\]]+)\]\s*(->|<-)\s*(\w+)\s*(\(inode (\d+), PID (\d+)\))?', message)


            if message_match:
                operation_code = message_match.group(1)
                location = message_match.group(2).strip()
                direction = message_match.group(3)
                operation = message_match.group(4)
                inode = None
                pid = None
                
                if message_match.group(5):
                    inode = message_match.group(6)
                    pid = message_match.group(7)
                
                if len(operation_ids) > 0 and operation_code == operation_ids[len(operation_ids)-1][1]:
                    response_ids.append((dt_object, operation_code))
                    
                if operation == "SyncFile":
                    operation_ids.append((dt_object, operation_code))

                return {
                    "time": dt_object,
                    "severity": severity,
                    "operation_code": operation_code,
                    "location": location,
                    "operation": operation,
                    "inode": inode,
                    "pid": pid
                }
            else:
              return {
                    "time": dt_object,
                    "severity": severity,
                    "message": message #If message format is different
                }



        except ValueError:
            print(f"Error parsing time: {time_str}")
            return None  # Or handle the error as needed

    return None  # Line doesn't match the expected pattern

def process_log_file(filename):
    """Reads a log file, parses each line, and prints the results.

    Args:
        filename: The path to the log file.
    """

    try:
        with open(filename, 'r') as f:
            for line_number, line in enumerate(f, 1):  # Enumerate for line numbers
                line = line.strip()  # Remove leading/trailing whitespace
                if line: # Skip empty lines
                    parsed_data = parse_log_line(line)
                    if parsed_data:
                        if debug:
                            print(f"Line {line_number}: {parsed_data}")
                            # Example: Reformat time and print specific fields
                            print(f"  Time: {parsed_data['time'].strftime('%Y-%m-%d %H:%M:%S.%f')}")
                            if 'operation' in parsed_data: # Check if operation is present (handles different message formats)
                                print(f"  Operation: {parsed_data['operation']}")
                                print(f"  Operation code: {parsed_data['operation_code']}")
                                print(f"  Inode: {parsed_data['inode']}")
                                print(f"  PID: {parsed_data['pid']}")
                    else:
                        print(f"Line {line_number}: Could not parse: {line}") # Print unparsed lines for debugging
    except FileNotFoundError:
        print(f"Error: File '{filename}' not found.")
    except Exception as e: # Catch any other exceptions during file reading
        print(f"An error occurred while reading the file: {e}")


log_file_path = "/home/princer_google_com/logs/gcsfuse.log"
process_log_file(log_file_path)

print(f" Len: {len(operation_ids)}")
print(f" Len: {len(response_ids)}")

if len(operation_ids) != len(response_ids):
    print("Len for op-code and response should be same")
    exit(0)

# Iterate over the both list with index
for i in range(len(operation_ids)):
    # print(f"Operation ID: {operation_ids[i][0]} {operation_ids[i][1]}, Response ID: {response_ids[i][0]} {response_ids[i][1]}")
    latency = response_ids[i][0] - operation_ids[i][0]
    if i < 1023:
        print(f" Time taken in the {i}th 1 MiB append: {latency.total_seconds()}s")
    # if i > 1023 and i < 2047:
    #     print(f" Time taken in the {i}th 1 MiB append: {latency.total_seconds()}s")
    # if i > 2047 and i < 3071:
    #     print(f" Time taken in the {i % 1024}th 1 MiB append: {latency.total_seconds()}s")
    # if i > 3071:
    #     print(f" Time taken in the {i%1024}th 1 MiB append: {latency.total_seconds()}s")



# import matplotlib.pyplot as plt
# import random
# import time

# def plot_random_time_graph():
#     x_values = []
#     y_values = []

#     for i in range(len(operation_ids)):
#         # if i < 1024:
#         # if i > 1024 and i < 2048:
#         # if i > 2048 and i < 3072:
#         if i > 3072:
#             x_values.append(i % 1024)
#             latency = response_ids[i][0] - operation_ids[i][0]
#             y_values.append(latency.total_seconds())


#     plt.figure(figsize=(12, 6))  # Adjust figure size as needed
#     plt.plot(x_values, y_values)  # Use plt.plot for a line graph

#     plt.xlabel("Offset in MiB (0-1024)")
#     plt.ylabel("SyncFile timining (secs)")
#     plt.title("Graph SyncFile timinig vs. append-offset")
#     plt.grid(True) # Add a grid for better readability

#     plt.xticks(range(0, 1025, 128))  # Set x-axis ticks every 128 units (adjust as needed)
#     plt.tight_layout()  # Adjust layout to prevent labels from overlapping
#     plt.savefig("zonal1.png")

# plot_random_time_graph()





