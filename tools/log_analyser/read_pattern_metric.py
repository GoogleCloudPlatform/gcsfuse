

def parseit(log):
    message = log["message"]
    # {"timestamp":{"seconds":1716900537,"nanos":713138380},"severity":"TRACE","message":"fuse_debug: Op 0x000033e0        connection.go:420] <- ReadFile (inode 6, PID 96407, handle 155, offset 192512, 4096 bytes)"}
    start_index = message.find("inode")+6
    end_index = message.find(",", start_index)
    inode = int(message[start_index:end_index])
    start_index = message.find("PID")+4
    end_index = message.find(",", start_index)
    pid = int(message[start_index:end_index])
    start_index = message.find("handle")+7
    end_index = message.find(",", start_index)
    handle = int(message[start_index:end_index])
    start_index = message.find("offset")+7
    end_index = message.find(",", start_index)
    offset = int(message[start_index:end_index])
    start_index = message.rfind(",")+2
    end_index = message.find(" ", start_index)
    byts = int(message[start_index:end_index])
    return [inode, pid, handle, offset, byts]


def processor(file, logs):
    req = ""
    inode = -1
    pattern = {}
    last_entry = {}
    for log in logs:
        # data = parseit(log)
        # if (inode == data[0]) and (handle == data[2]):
        #     # for i in range(5):
        #     #     print(data[i], " ")
        #     # print("\n")
        #     if last_entry == -1:
        #         pattern += "_"
        #         last_entry = data[3] + data[4]
        #     else:
        #         if data[3] == last_entry:
        #             pattern += "s"
        #         else:
        #             pattern += "r"
        #         last_entry = data[3] + data[4]
        message = log["message"]
        if message.find("ReadFile") != -1:
            data = parseit(log)
            if inode == data[0]:
                if data[2] not in pattern:
                    pattern[data[2]] = ""
                    pattern[data[2]] += "_"
                    last_entry[data[2]] = data[3] + data[4]
                else:
                    if data[3] == last_entry[data[2]]:
                        pattern[data[2]] += "s"
                    else:
                        pattern[data[2]] += "r"
                    last_entry[data[2]] = data[3] + data[4]
        else:
            if message.find("LookUpInode") != -1 and message.find(file) != -1:
                start_index = message.find("Op 0x")+3
                end_index = start_index+10
                req = message[start_index:end_index]
            if message.find("OK (inode") != -1:
                start_index = message.find("Op 0x")+3
                end_index = start_index+10
                if req == message[start_index:end_index]:
                    start_index = message.find("(inode ")+7
                    end_index = message.rfind(")")
                    inode = int(message[start_index:end_index])

    for handle in pattern.keys():
        print("Pattern of reads for handle = ", handle, ":", pattern[handle], "\n")
        # pattern = "_rrrssssrr"
        # assuming total reads >1
        if len(pattern[handle]) > 1:
            last_read = pattern[handle][1]
            streak = 1
            for i in range(2, len(pattern[handle])):
                if pattern[handle][i] != last_read:
                    print(last_read, streak, " ")
                    last_read = pattern[handle][i]
                    streak = 1
                else:
                    streak += 1

            print(last_read, streak, "\n")
        else:
            print("A single read happened\n")

    if len(pattern) == 0:
        print("No reads happened for the given file\n")





