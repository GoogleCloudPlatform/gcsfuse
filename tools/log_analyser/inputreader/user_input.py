from inputreader.get_logs import GetLogs
class UserInput:
    def get_input(self):
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
        get_logs_obj = GetLogs()
        log_type = input("Enter the type of logs (gcsfuse/gke): ")
        logs = get_logs_obj.get_json_logs(files, log_type)
        return logs
