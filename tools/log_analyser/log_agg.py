import json
from input import open_file


def open_json(file):
    try:
        with open(file, "r") as json_ref:
            # logs = []
            logs = json.load(json_ref)
            # for line in json_ref:
            # logs.append(json.load(line))
            return logs
    except (FileNotFoundError, json.JSONDecodeError) as e:
        print(f"Error reading JSON log file: {e}")
        return None


def seg_log(files, agg_map):
    # agg_file_ref = []
    # for file in agg_file:
    #   agg_file_ref.append(open_file(file,"w"))
    file_map = {}
    request_map = {}
    for file in files:
        logs = open_json(file)
        if logs:
            for log in logs:
                message = log["message"]

                if message.find("OpenFile") != -1:
                    # json.dump(log, agg_file_ref[0])
                    # add to more files
                    agg_map[0].append(log)

                if message.find("LookUpInode") != -1:
                    start_index = message.find("name ") + 6
                    end_index = message.rfind("\"")
                    name = message[start_index:end_index]
                    if name not in file_map:
                      start_index = message.find("Op 0x") + 3
                      end_index = start_index + 10
                      req = message[start_index:end_index]
                      request_map[req] = name

                if message.find("OK (inode") != -1:
                    start_index = message.find("Op 0x") + 3
                    end_index = start_index + 10
                    req = message[start_index:end_index]
                    if req in request_map:
                        start_index = message.find("(inode ") + 7
                        end_index = message.rfind(")")
                        inode = message[start_index:end_index]
                        file_map[inode] = request_map[req]
                        request_map.pop(req)

                if message.find("ReadFile") != -1:
                    agg_map[1].append(log)
                # for other logs add more ifs
    return file_map
