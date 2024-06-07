# put this function in utility file
def get_val(message, key, delim, direction, offset):
    # offset contains adjustments needed for spaces and key lengths
    if direction == "fwd":
        start_index = message.find(key)+len(key)+offset
    else:
        start_index = message.rfind(key)+len(key)+offset
    end_index = message.find(delim, start_index)
    return message[start_index:end_index]


def parseit(log):
    message = log["message"]
    inode = int(get_val(message, "inode", ",", "fwd", 1))
    pid = int(get_val(message, "PID", ",", "fwd", 1))
    handle = int(get_val(message, "handle", ",", "fwd", 1))
    offset = int(get_val(message, "offset", ",", "fwd", 1))
    byts = int(get_val(message, ",", " ", "bck", 1))
    return [inode, pid, handle, offset, byts]


def processor(file, logs):
    req = ""
    inode = -1
    pattern = {}
    avg_read_size = {}
    last_entry = {}
    for log in logs:
        message = log["message"]
        if message.find("ReadFile") != -1:
            data = parseit(log)
            if inode == data[0]:
                if data[2] not in pattern:
                    pattern[data[2]] = ""
                    pattern[data[2]] += "_"
                    last_entry[data[2]] = data[3] + data[4]
                    avg_read_size[data[2]] = 0
                else:
                    if data[3] == last_entry[data[2]]:
                        pattern[data[2]] += "s"
                    else:
                        pattern[data[2]] += "r"
                    last_entry[data[2]] = data[3] + data[4]
                avg_read_size[data[2]] += data[4]

        else:
            if message.find("LookUpInode") != -1 and message.find(file) != -1:
                req = get_val(message, "Op 0x", " ", "fwd", -1)
            if message.find("OK (inode") != -1:
                if req == get_val(message, "Op 0x", " ", "fwd", -1):
                    inode = int(get_val(message, "(inode", ")", "fwd", 1))

    for handle in pattern.keys():
        print("Pattern of reads for handle = ", handle, ":")
        if len(pattern[handle]) > 1:
            last_read = pattern[handle][1]
            streak = 1
            for i in range(2, len(pattern[handle])):
                if pattern[handle][i] != last_read:
                    print(last_read, streak, end="\t")
                    last_read = pattern[handle][i]
                    streak = 1
                else:
                    streak += 1

            print(last_read, streak)
            print("Total reads for this handle:", len(pattern[handle]))
            print("Average read size:", avg_read_size[handle]/len(pattern[handle]))
        else:
            print("A single read happened\n")
            print("Read size:", avg_read_size[handle])

    if len(pattern) == 0:
        print("No reads happened for the given file\n")





