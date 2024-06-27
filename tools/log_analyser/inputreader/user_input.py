from inputreader.get_logs import GetLogs
import os
class UserInput:
    def get_input(self):
        files = []
        directory_path = input("Enter the path to the directory containing log files: ")
        for filename in os.listdir(directory_path):
            # Construct the full path to the file
            file_path = os.path.join(directory_path, filename)

            # Check if it's a regular file (not a directory or hidden file)
            if os.path.isfile(file_path):
                files.append(file_path)

        add_time_filter = input("Do you want the time filter(y/n):" )
        if add_time_filter == "y" or add_time_filter == "Y":
            start_time = int(input("start time(epoch): "))
            end_time = int(input("end time(epoch): "))
        else:
            start_time = 0
            end_time = 1e18
        get_logs_obj = GetLogs()
        log_type = input("Enter the type of logs (gcsfuse/gke): ")
        logs = get_logs_obj.get_json_logs(files, log_type, [start_time, end_time])
        return logs
