import json


def seg_log(files, agg_map, start_time, end_time, filename):
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
    if logs:
        for log in logs:
            message = log["message"]
            timestamp = int(log["timestamp"]["seconds"])
            if timestamp < start_time:
                continue
            if timestamp > end_time:
                break

            if message.find("OpenFile") != -1:
                agg_map[0].append(log)

            if message.find("LookUpInode") != -1:
                agg_map[0].append(log)
                agg_map[1].append(log)

            if message.find("OK (inode") != -1:
                agg_map[0].append(log)
                agg_map[1].append(log)

            if message.find("ReadFile") != -1:
                agg_map[1].append(log)
            if message.find(filename) != -1:
                agg_map[2].append(log)
            # for other logs add more ifs

    return logs

