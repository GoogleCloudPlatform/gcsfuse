from parser.object import Object
from parser.handle import Handle
from parser.global_data import GlobalData
from parser.requests import Request


def get_val(message, key, delim, direction, offset):
    # offset contains adjustments needed for spaces and key lengths
    try:
        if message.find(key) == -1:
            print("Error parsing log with message:", message)
            return None
        if direction == "fwd":
            start_index = message.find(key)+len(key)+offset
        else:
            start_index = message.rfind(key)+len(key)+offset
        if delim != "end_line":
            if message.find(delim, start_index) == -1:
                print("Error parsing log with message:", message)
                return None
            end_index = message.find(delim, start_index)
        else:
            end_index = len(message) - 1
        return message[start_index:end_index]
    except ValueError as e:
        print("Error parsing log with message:", message)
        return None


def lookup_processor(log, global_data):
    message = log["message"]
    req_id = get_val(message, "Op 0x", " ", "fwd", 0)
    name = get_val(message, "name", "\"", "fwd", 2)
    parent_tmp = get_val(message, "parent", ",", "fwd", 1)
    if parent_tmp is None or name is None or req_id is None:
        return
    try:
        parent = int(parent_tmp)
    except ValueError as e:
        print("Error parsing parent:", parent_tmp)
        return
    if parent != 0 and parent != 1 and parent in global_data.inode_name_map:
        prefix_name = global_data.inode_name_map[parent]
        prefix_name += "/"
    else:
        prefix_name = ""
    abs_name = prefix_name + name
    global_data.requests[req_id] = Request("LookUpInode", abs_name)
    request_obj = global_data.requests[req_id]
    if abs_name in global_data.name_object_map.keys():
        global_data.name_object_map[abs_name].kernel_calls.calls[0].calls_made += 1
        request_obj.is_lookup_call_added = True
    if parent not in global_data.inode_name_map:
        request_obj.is_valid = False
    request_obj.parent = parent
    request_obj.rel_name = name
    global_data.gcalls.kernel_calls[6].calls_made += 1
    request_obj.timestamp_sec = log["timestamp"]["seconds"]
    request_obj.timestamp_nano = log["timestamp"]["nanos"]


def gcs_call_processor(log, global_data):
    message = log["message"]
    name = get_val(message, "(", "\"", "fwd", 1)
    if name is None:
        return
    if name not in global_data.name_object_map.keys():
        global_data.name_object_map[name] = Object(None, None, None, name)
    if len(name) > 0 and name[len(name)-1] == "/":
        global_data.name_object_map[name].is_dir = True

    if message.find("<-") != -1:
        req = get_val(message, "0x", " ", "fwd", 0)
        req_name = get_val(message, "<-", "(", "fwd", 1)
        if req is None or req_name not in global_data.gcalls.gcs_index_map.keys():
            return
        global_data.requests[req] = Request("gcsreq", name)
        global_data.requests[req].timestamp_sec = log["timestamp"]["seconds"]
        global_data.requests[req].timestamp_nano = log["timestamp"]["nanos"]
        global_data.requests[req].keyword = req_name
        file_obj = global_data.name_object_map[name]
        file_obj.gcs_calls.calls[file_obj.gcs_calls.callname_index_map[req_name]].calls_made += 1
        global_data.gcalls.gcs_calls[global_data.gcalls.gcs_index_map[req_name]].calls_made += 1
        if message.find("Read") != -1 and message.find("nil") == -1:
            start_temp = get_val(message, "[", ",", "fwd", 0)
            final_temp = get_val(message, ",", ")", "bck", 1)
            if start_temp is None or final_temp is None:
                return
            try:
                start = int(start_temp)
                final = int(final_temp)
            except ValueError as e:
                print("Error parsing bytes:", start_temp, "or", final_temp)
                return
            global_data.bytes_from_gcs += final-start

    elif message.find("->") != -1:
        req = get_val(message, "0x", " ", "fwd", 0)
        req_name = get_val(message, "->", "(", "fwd", 1)
        if req is None or req_name not in global_data.gcalls.gcs_index_map.keys():
            return

        if req in global_data.requests.keys():
            call_obj = global_data.name_object_map[name].gcs_calls.calls[global_data.name_object_map[name].gcs_calls.callname_index_map[req_name]]
            global_call_obj = global_data.gcalls.gcs_calls[global_data.gcalls.gcs_index_map[req_name]]
            call_obj.calls_returned += 1
            global_call_obj.calls_returned += 1

            time_sec = log["timestamp"]["seconds"]
            time_nano = log["timestamp"]["nanos"]
            req_response_time = 1e3*(time_sec + 1e-9*time_nano - global_data.requests[req].timestamp_sec - 1e-9*global_data.requests[req].timestamp_nano)
            call_obj.total_response_time += req_response_time
            call_obj.response_times.append(req_response_time)
            global_call_obj.total_response_time += req_response_time
            global_call_obj.response_times.append(req_response_time)
            if message.find("StatObject") != -1:
                if message.find("gcs.NotFoundError: storage: object doesn't exist") != -1:
                    global_data.name_object_map[name].is_dir = True

            global_data.requests.pop(req)


def open_file_processor(log, global_data):
    message = log["message"]
    inode_temp = get_val(message, "inode", ",", "fwd", 1)
    req_id = get_val(message, "Op 0x", " ", "fwd", 0)
    if req_id is None or inode_temp is None:
        return
    try:
        inode = int(inode_temp)
    except ValueError as e:
        print("Error parsing inode:", inode_temp)
        return
    if inode in global_data.inode_name_map.keys():
        global_data.requests[req_id] = Request("OpenFile", global_data.inode_name_map[inode])
        global_data.name_object_map[global_data.inode_name_map[inode]].kernel_calls.calls[2].calls_made += 1
    else:
        global_data.requests[req_id] = Request("OpenFile", "")
        global_data.requests[req_id].is_valid = False
    global_data.requests[req_id].inode = inode
    global_data.requests[req_id].timestamp_sec = log["timestamp"]["seconds"]
    global_data.requests[req_id].timestamp_nano = log["timestamp"]["nanos"]
    global_data.gcalls.kernel_calls[8].calls_made += 1


def release_file_handle_processor(log, global_data):
    message = log["message"]
    handle_temp = get_val(message, "handle", ")", "fwd", 1)
    if handle_temp is not None:
        try:
            handle = int(handle_temp)
        except ValueError as e:
            print("Error parsing handle:", handle_temp)
            return
    else:
        return
    req_id = get_val(message, "Op 0x", " ", "fwd", 0)
    if req_id is None:
        return
    global_data.gcalls.kernel_calls[13].calls_made += 1
    global_data.requests[req_id] = Request("ReleaseFileHandle", "")
    global_data.requests[req_id].timestamp_sec = log["timestamp"]["seconds"]
    global_data.requests[req_id].timestamp_nano = log["timestamp"]["nanos"]
    if handle is not None and handle in global_data.handle_name_map.keys():
        global_data.requests[req_id].object_name = global_data.handle_name_map[handle]
        obj = global_data.name_object_map[global_data.handle_name_map[handle]]
        obj.closed_handles += 1
        obj.handles[handle].closing_time = log["timestamp"]["seconds"]
        obj.handles[handle].closing_time_nano = log["timestamp"]["nanos"]
        obj.handles[handle].close_pos = obj.closed_handles
        obj.close_tup.append([[int(log["timestamp"]["seconds"]), int(log["timestamp"]["nanos"])], obj.closed_handles])
        obj.kernel_calls.calls[7].calls_made += 1


def read_file_processor(log, global_data):
    message = log["message"]
    inode_temp = get_val(message, "inode", ",", "fwd", 1)
    handle_temp = get_val(message, "handle", ",", "fwd", 1)
    offset_temp = get_val(message, "offset", ",", "fwd", 1)
    byts_temp = get_val(message, ",", " ", "bck", 1)
    req_id = get_val(message, "Op 0x", " ", "fwd", 0)
    if req_id is None or byts_temp is None or offset_temp is None or handle_temp is None or inode_temp is None:
        return
    try:
        inode = int(inode_temp)
        handle = int(handle_temp)
        offset = int(offset_temp)
        byts = int(byts_temp)
    except ValueError as e:
        print("Error parsing:", inode_temp, handle_temp, offset_temp, byts_temp)
        return
    if inode in global_data.inode_name_map.keys():
        obj = global_data.name_object_map[global_data.inode_name_map[inode]]
        if handle not in obj.handles:
            obj.handles[handle] = Handle(handle, 0, 0)
            global_data.handle_name_map[handle] = global_data.inode_name_map[inode]
        handle_obj = obj.handles[handle]
        if message.find("ReadFile") != -1:
            global_data.gcalls.kernel_calls[7].calls_made += 1
            global_data.requests[req_id] = Request("ReadFile", global_data.inode_name_map[inode])
            if handle_obj.last_read_offset == -1:
                handle_obj.read_pattern += "_"
            elif handle_obj.last_read_offset == offset:
                handle_obj.read_pattern += "s"
            else:
                handle_obj.read_pattern += "r"
            handle_obj.last_read_offset = offset + byts
            handle_obj.total_read_size += byts
            handle_obj.total_reads += 1
            obj.kernel_calls.calls[1].calls_made += 1

        else:
            global_data.gcalls.kernel_calls[10].calls_made += 1
            global_data.requests[req_id] = Request("WriteFile", global_data.inode_name_map[inode])
            handle_obj.total_writes += 1
            handle_obj.total_write_size += byts
            global_data.bytes_to_gcs += byts
            obj.kernel_calls.calls[4].calls_made += 1
    else:
        if message.find("ReadFile") != -1:
            global_data.requests[req_id] = Request("ReadFile", "")
            global_data.gcalls.kernel_calls[7].calls_made += 1
        else:
            global_data.requests[req_id] = Request("WriteFile", "")
            global_data.gcalls.kernel_calls[10].calls_made += 1
            global_data.bytes_to_gcs += byts
        global_data.requests[req_id].is_valid = False
    global_data.requests[req_id].inode = inode
    global_data.requests[req_id].handle = handle
    global_data.requests[req_id].timestamp_sec = log["timestamp"]["seconds"]
    global_data.requests[req_id].timestamp_nano = log["timestamp"]["nanos"]


def kernel_call_processor(log, global_data):
    message = log["message"]
    if message.find("(") == -1:
        return
    name = None
    req_id = get_val(message, "Op 0x", " ", "fwd", 0)
    req_name = get_val(message, "<-", " ", "fwd", 1)
    if req_id is None or req_name is None:
        return
    global_data.requests[req_id] = Request(req_name, "")
    global_data.requests[req_id].timestamp_sec = log["timestamp"]["seconds"]
    global_data.requests[req_id].timestamp_nano = log["timestamp"]["nanos"]
    if message.find("inode") != -1:
        inode_temp = get_val(message, "inode", ",", "fwd", 1)
        if inode_temp is None:
            return
        inode = int(inode_temp)
        if inode in global_data.inode_name_map.keys():
            global_data.requests[req_id].object_name = global_data.inode_name_map[inode]
            file_obj = global_data.name_object_map[global_data.inode_name_map[inode]]
            if req_name in file_obj.kernel_calls.callname_index_map.keys():
                file_obj.kernel_calls.calls[file_obj.kernel_calls.callname_index_map[req_name]].calls_made += 1
    elif message.find("name") != -1:
        name = get_val(message, "name", "\"", "fwd", 2)
        if name is None:
            return
        if name in global_data.name_object_map.keys():
            global_data.requests[req_id].object_name = name
            file_obj = global_data.name_object_map[name]
            if req_name in file_obj.kernel_calls.callname_index_map.keys():
                file_obj.kernel_calls.calls[file_obj.kernel_calls.callname_index_map[req_name]].calls_made += 1
    if req_name in global_data.gcalls.kernel_index_map.keys():
        global_data.gcalls.kernel_calls[global_data.gcalls.kernel_index_map[req_name]].calls_made += 1
    if message.find("CreateFile") != -1:
        parent_temp = get_val(message, "parent", ",", "fwd", 1)
        if parent_temp is None:
            return
        try:
            parent = int(parent_temp)
        except ValueError as e:
            print("Error parsing parent:", parent_temp)
            return
        if parent in global_data.inode_name_map:
            prefix = global_data.inode_name_map[parent] + "/"
        else:
            prefix = ""
            global_data.requests[req_id].is_valid = False
        global_data.requests[req_id].abs_name = prefix + name
        global_data.requests[req_id].rel_name = name
        global_data.requests[req_id].parent = parent


def response_processor(log, global_data):
    message = log["message"]
    time_sec = log["timestamp"]["seconds"]
    time_nano = log["timestamp"]["nanos"]
    req_id = get_val(message, "Op 0x", " ", "fwd", 0)
    if req_id is None:
        return
    if req_id in global_data.requests.keys():
        req = global_data.requests[req_id]
        req_name = req.req_type
        if req_name in global_data.gcalls.kernel_index_map.keys():
            global_data.gcalls.kernel_calls[global_data.gcalls.kernel_index_map[req_name]].calls_returned += 1
        if req.is_valid:
            if req.object_name in global_data.name_object_map.keys():
                obj = global_data.name_object_map[req.object_name]
            else:
                global_data.name_object_map[req.object_name] = Object(req.inode, req.parent, "", req.object_name)
                obj = global_data.name_object_map[req.object_name]
            if req_name in obj.kernel_calls.callname_index_map.keys():
                obj.kernel_calls.calls[obj.kernel_calls.callname_index_map[req_name]].calls_returned += 1
            if req_name == "ReadFile":
                obj.handles[req.handle].read_times.append(time_sec + 1e-9*time_nano - req.timestamp_sec - 1e-9*req.timestamp_nano)
            elif req_name == "WriteFile":
                obj.handles[req.handle].write_times.append(time_sec + 1e-9*time_nano - req.timestamp_sec - 1e-9*req.timestamp_nano)
            elif req_name == "LookUpInode" or req_name == "CreateFile":
                if message.find("Error") != -1:
                    obj.is_dir = False
                    global_data.requests.pop(req_id)
                    return
                inode_temp = get_val(message, "(inode", ")", "fwd", 1)
                if inode_temp is None:
                    return
                try:
                    inode = int(inode_temp)
                except ValueError as e:
                    print("Error parsing inode:", inode_temp)
                    return
                if not req.is_lookup_call_added and req.req_type == "LookUpInode":
                    obj.kernel_calls.calls[0].calls_made += 1
                obj.inode = inode
                obj.parent = req.parent
                obj.rel_name = req.rel_name
                obj.abs_name = req.object_name
                global_data.inode_name_map[inode] = req.object_name

            elif req_name == "OpenFile":
                handle_temp = get_val(message, "handle", ")", "fwd", 1)
                if handle_temp is not None:
                    try:
                        handle = int(handle_temp)
                    except ValueError as e:
                        print("Error parsing handle:", handle_temp)
                        return
                    obj.handles[handle] = Handle(handle, req.timestamp_sec, req.timestamp_nano)
                    obj.opened_handles += 1
                    obj.handles[handle].open_pos = obj.opened_handles
                    obj.open_tup.append([[int(req.timestamp_sec), int(req.timestamp_nano)], obj.opened_handles])
                    global_data.handle_name_map[handle] = req.object_name

        global_data.requests.pop(req_id)


def gen_processor(logs):
    global_data = GlobalData()
    global_data.name_object_map[""] = Object(1, 0, "", "")
    global_data.name_object_map[""].is_dir = True
    global_data.inode_name_map[1] = ""
    for log in logs:
        message = log["message"]
        if message.find("LookUpInode") != -1 and message.find("fuse_debug") != -1 and message.find("<-") != -1:
            lookup_processor(log, global_data)
        elif message.find("gcs: Req") != -1:
            gcs_call_processor(log, global_data)
        elif message.find("OpenFile") != -1 and message.find("fuse_debug") != -1 and message.find("<-") != -1:
            open_file_processor(log, global_data)
        elif message.find("ReleaseFileHandle") != -1 and message.find("fuse_debug") != -1 and message.find("<-") != -1:
            release_file_handle_processor(log, global_data)
        elif (message.find("ReadFile") != -1 or message.find("WriteFile") != -1) and message.find("fuse_debug") != -1 and message.find("<-") != -1:
            read_file_processor(log, global_data)
        elif message.find("fuse_debug") != -1 and message.find("<-") != -1:
            kernel_call_processor(log, global_data)
        elif message.find("fuse_debug") != -1 and message.find("->") != -1:
            response_processor(log, global_data)

    return global_data
