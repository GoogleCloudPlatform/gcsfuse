import utility
from classes import Object as Object
from classes import Handle as Handle
from classes import GlobalData as GlobalData
from classes import Request as Request


def lookup_processor(log, global_data):
    message = log["message"]
    req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
    name = utility.get_val(message, "name", "\"", "fwd", 2)
    parent_tmp = utility.get_val(message, "parent", ",", "fwd", 1)
    if parent_tmp is None or name is None or req_id is None:
        return
    try:
        parent = int(parent_tmp)
    except ValueError as e:
        print("Error parsing parent:", parent_tmp)
        return
    if parent != 0 and parent != 1 and parent in global_data.inode_name_map:
        # give_dir_tag(global_data, parent)
        prefix_name = global_data.inode_name_map[parent]
        prefix_name += "/"
    else:
        prefix_name = ""
    abs_name = prefix_name + name
    global_data.requests[req_id] = Request("LookUpInode", abs_name)
    if abs_name in global_data.name_object_map.keys():
        global_data.name_object_map[abs_name].kernel_calls.calls[0].calls_made += 1
        global_data.requests[req_id].is_lookup_call_added = True
    if parent not in global_data.inode_name_map:
        global_data.requests[req_id].is_valid = False
    global_data.requests[req_id].parent = parent
    global_data.requests[req_id].rel_name = name
    global_data.gcalls.kernel_calls[6].calls_made += 1
    global_data.requests[req_id].timestamp_sec = log["timestamp"]["seconds"]
    global_data.requests[req_id].timestamp_nano = log["timestamp"]["nanos"]


def response_processor(log, global_data):
    message = log["message"]
    time_sec = log["timestamp"]["seconds"]
    time_nano = log["timestamp"]["nanos"]
    req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
    if req_id is None:
        return
    if req_id in global_data.requests.keys():
        req = global_data.requests[req_id]
        if req.is_valid:
            if req.object_name in global_data.name_object_map.keys():
                obj = global_data.name_object_map[req.object_name]
            else:
                global_data.name_object_map[req.object_name] = Object(req.inode, req.parent, "", req.object_name)
                obj = global_data.name_object_map[req.object_name]
                # if req.req_type == "LookUpInode":
                #     obj.kernel_calls.calls[0].calls_made += 1


        if req.req_type == "LookUpInode":
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[6], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])
            if req.is_valid:
                if message.find("Error") != -1:
                    if req.object_name in global_data.name_object_map.keys():
                        obj.is_dir = False
                        global_data.requests.pop(req_id)
                        return
                inode_temp = utility.get_val(message, "(inode", ")", "fwd", 1)
                if inode_temp is None:
                    return
                try:
                    inode = int(inode_temp)
                except ValueError as e:
                    print("Error parsing inode:", inode_temp)
                    return
                if not req.is_lookup_call_added:
                    obj.kernel_calls.calls[0].calls_made += 1
                obj.inode = inode
                obj.parent = req.parent
                obj.rel_name = req.rel_name
                obj.abs_name = req.object_name
                obj.kernel_calls.calls[0].calls_returned += 1
                global_data.inode_name_map[inode] = req.object_name

        elif req.req_type == "OpenFile":
            if req.is_valid:
                handle_temp = utility.get_val(message, "handle", ")", "fwd", 1)
                if handle_temp is not None:
                    try:
                        handle = int(handle_temp)
                    except ValueError as e:
                        print("Error parsing handle:", handle)
                    obj.handles[handle] = Handle(handle, req.timestamp_sec, req.timestamp_nano)
                    obj.opened_handles += 1
                    obj.open_tup.append([[int(req.timestamp_sec), int(req.timestamp_nano)], obj.opened_handles])
                    obj.kernel_calls.calls[2].calls_returned += 1
                    global_data.handle_name_map[handle] = req.object_name
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[8], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "CreateFile":
            if req.is_valid:
                inode_tmp = utility.get_val(message, "inode", ")", "fwd", 1)
                if inode_tmp is None:
                    return
                try:
                    inode = int(inode_tmp)
                except ValueError as e:
                    print("Error parsing inode:", inode_tmp)
                    return
                obj.inode = inode
                obj.parent = req.parent
                obj.abs_name = req.object_name
                obj.rel_name = req.rel_name
                global_data.inode_name_map[inode] = obj.abs_name
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[4], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "ReadFile":
            if req.is_valid:
                obj.kernel_calls.calls[1].calls_returned += 1
                obj.handles[req.handle].read_times.append(time_sec + 1e-9*time_nano - req.timestamp_sec - 1e-9*req.timestamp_nano)
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[7], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "WriteFile":
            if req.is_valid:
                obj.kernel_calls.calls[4].calls_returned += 1
                obj.handles[req.handle].write_times.append(time_sec + 1e-9*time_nano - req.timestamp_sec - 1e-9*req.timestamp_nano)
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[10], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "ReadDir":
            if req.is_valid:
                obj.kernel_calls.calls[9].calls_returned += 1
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[15], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "OpenDir":
            if req.is_valid:
                obj.kernel_calls.calls[8].calls_returned += 1
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[14], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "ReleaseFileHandle":
            if req.is_valid:
                obj.kernel_calls.calls[7].calls_returned += 1

            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[13], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "ReadSymLink":
            if req.is_valid:
                obj.kernel_calls.calls[6].calls_returned += 1
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[12], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "CreateSymLink":
            if req.is_valid:
                obj.kernel_calls.calls[5].calls_returned += 1
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[11], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "FlushFile":
            if req.is_valid:
                obj.kernel_calls.calls[3].calls_returned += 1
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[9], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "RmDir":
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[5], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "ReleaseDirHandle":
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[3], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "MkDir":
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[2], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "Rename":
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[1], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        elif req.req_type == "Unlink":
            utility.update_global_kernel_calls(global_data.gcalls.kernel_calls[0], [time_sec, time_nano], [req.timestamp_sec, req.timestamp_nano])

        global_data.requests.pop(req_id)


def gcs_call_processor(log, global_data):
    message = log["message"]
    name = utility.get_val(message, "(", "\"", "fwd", 1)
    if name is None:
        return
    if name not in global_data.name_object_map.keys():
        global_data.name_object_map[name] = Object(None, None, None, name)
    if len(name) > 0 and name[len(name)-1] == "/":
        global_data.name_object_map[name].is_dir = True

    if message.find("StatObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[0]
        global_call_obj = global_data.gcalls.gcs_calls[0]
        if message.find("gcs.NotFoundError: storage: object doesn't exist") != -1:
            global_data.name_object_map[name].is_dir = True

    elif message.find("ListObjects") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[1]
        global_call_obj = global_data.gcalls.gcs_calls[1]

    elif message.find("CopyObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[2]
        global_call_obj = global_data.gcalls.gcs_calls[2]

    elif message.find("ComposeObjects") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[3]
        global_call_obj = global_data.gcalls.gcs_calls[3]

    elif message.find("UpdateObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[4]
        global_call_obj = global_data.gcalls.gcs_calls[4]

    elif message.find("DeleteObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[5]
        global_call_obj = global_data.gcalls.gcs_calls[5]

    elif message.find("CreateObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[6]
        global_call_obj = global_data.gcalls.gcs_calls[6]

    elif message.find("Read") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[7]
        global_call_obj = global_data.gcalls.gcs_calls[7]
        if message.find("<-") != -1 and message.find("nil") == -1:
            start_temp = utility.get_val(message, "[", ",", "fwd", 0)
            final_temp = utility.get_val(message, ",", ")", "bck", 1)
            if start_temp is None or final_temp is None:
                return
            try:
                start = int(start_temp)
                final = int(final_temp)
            except ValueError as e:
                print("Error parsing bytes:", start_temp, "or", final_temp)
                return
            global_data.bytes_from_gcs += final-start

    if message.find("<-") != -1:
        req = utility.get_val(message, "0x", " ", "fwd", 0)
        if req is None:
            return
        global_data.requests[req] = Request("gcsreq", name)
        global_data.requests[req].timestamp_sec = log["timestamp"]["seconds"]
        global_data.requests[req].timestamp_nano = log["timestamp"]["nanos"]
        global_data.requests[req].keyword = call_obj.call_name
        call_obj.calls_made += 1
        global_call_obj.calls_made += 1

    elif message.find("->") != -1:
        req = utility.get_val(message, "0x", " ", "fwd", 0)
        if req is None:
            return

        if req in global_data.requests.keys():
            call_obj.calls_returned += 1
            global_call_obj.calls_returned += 1
            start_index = message.rfind("(")

            time_sec = log["timestamp"]["seconds"]
            time_nano = log["timestamp"]["nanos"]
            req_response_time = 1e3*(time_sec + 1e-9*time_nano - global_data.requests[req].timestamp_sec - 1e-9*global_data.requests[req].timestamp_nano)
            call_obj.total_response_time += req_response_time
            call_obj.response_times.append(req_response_time)
            global_call_obj.total_response_time += req_response_time
            global_call_obj.response_times.append(req_response_time)

            global_data.requests.pop(req)


def open_file_processor(log, global_data):
    message = log["message"]
    inode_temp = utility.get_val(message, "inode", ",", "fwd", 1)
    req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
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
    handle = int(utility.get_val(message, ", handle", ")", "fwd", 1))
    req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
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
        obj.close_tup.append([[int(log["timestamp"]["seconds"]), int(log["timestamp"]["nanos"])], obj.closed_handles])
        obj.kernel_calls.calls[7].calls_made += 1


def read_file_processor(log, global_data):
    message = log["message"]
    inode_temp = utility.get_val(message, "inode", ",", "fwd", 1)
    handle_temp = utility.get_val(message, "handle", ",", "fwd", 1)
    offset_temp = utility.get_val(message, "offset", ",", "fwd", 1)
    byts_temp = utility.get_val(message, ",", " ", "bck", 1)
    req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
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
    if inode in global_data.inode_name_map:
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
    inode = -1
    name = None
    req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
    req_name = utility.get_val(message, "<-", " ", "fwd", 1)
    if req_id is None or req_name is None:
        return
    global_data.requests[req_id] = Request(req_name, "")
    global_data.requests[req_id].timestamp_sec = log["timestamp"]["seconds"]
    global_data.requests[req_id].timestamp_nano = log["timestamp"]["nanos"]
    if message.find("inode") != -1:
        inode_temp = utility.get_val(message, "inode", ",", "fwd", 1)
        if inode_temp is None:
            return
        inode = int(inode_temp)
    elif message.find("name") != -1:
        name = utility.get_val(message, "name", "\"", "fwd", 2)
        if name is None:
            return

    if message.find("FlushFile") != -1:
        if inode in global_data.inode_name_map:
            global_data.name_object_map[global_data.inode_name_map[inode]].kernel_calls.calls[3].calls_made += 1
            global_data.requests[req_id].object_name = global_data.inode_name_map[inode]
        global_data.gcalls.kernel_calls[9].calls_made += 1
    elif message.find("ReadDir") != -1:
        if inode in global_data.inode_name_map:
            global_data.name_object_map[global_data.inode_name_map[inode]].kernel_calls.calls[9].calls_made += 1
            global_data.requests[req_id].object_name = global_data.inode_name_map[inode]
        global_data.gcalls.kernel_calls[15].calls_made += 1
    elif message.find("OpenDir") != -1:
        if inode in global_data.inode_name_map:
            global_data.name_object_map[global_data.inode_name_map[inode]].kernel_calls.calls[8].calls_made += 1
            global_data.requests[req_id].object_name = global_data.inode_name_map[inode]
        global_data.gcalls.kernel_calls[14].calls_made += 1
    elif message.find("ReadSymLink") != -1:
        if inode in global_data.inode_name_map:
            global_data.name_object_map[global_data.inode_name_map[inode]].kernel_calls.calls[6].calls_made += 1
            global_data.requests[req_id].object_name = global_data.inode_name_map[inode]
        global_data.gcalls.kernel_calls[12].calls_made += 1
    elif message.find("CreateSymLink") != -1:
        if name in global_data.name_object_map:
            global_data.name_object_map[name].kernel_calls.calls[5].calls_made += 1
            global_data.requests[req_id].object_name = name
        global_data.gcalls.kernel_calls[11].calls_made += 1
    elif message.find("Unlink") != -1:
        global_data.gcalls.kernel_calls[0].calls_made += 1
    elif message.find("Rename") != -1:
        global_data.gcalls.kernel_calls[1].calls_made += 1
    elif message.find("RmDir") != -1:
        global_data.gcalls.kernel_calls[5].calls_made += 1
    elif message.find("MkDir") != -1:
        global_data.gcalls.kernel_calls[2].calls_made += 1
    elif message.find("ReleaseDirHandle") != -1:
        global_data.gcalls.kernel_calls[3].calls_made += 1
    elif message.find("CreateFile") != -1:
        name = utility.get_val(message, "name", "\"", "fwd", 2)
        parent_temp = utility.get_val(message, "parent", ",", "fwd", 1)
        if name is None or parent_temp is None:
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
        # req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
        # global_data.requests[req_id] = Request("createfile", prefix + name)
        global_data.requests[req_id].abs_name = prefix + name
        global_data.requests[req_id].rel_name = name
        global_data.requests[req_id].parent = parent
        global_data.gcalls.kernel_calls[4].calls_made += 1



def gen_processor(logs):
    global_data = GlobalData()
    global_data.name_object_map[""] = Object(1, 0, "", "")
    global_data.name_object_map[""].is_dir = True
    global_data.inode_name_map[1] = ""
    # process_obj = Processor()
    for log in logs:
        message = log["message"]
        if message.find("LookUpInode") != -1:
            lookup_processor(log, global_data)
        elif message.find("gcs: Req") != -1:
            gcs_call_processor(log, global_data)
        elif message.find("OpenFile") != -1:
            open_file_processor(log, global_data)
        elif message.find("ReleaseFileHandle") != -1:
            release_file_handle_processor(log, global_data)
        elif message.find("ReadFile") != -1 or message.find("WriteFile") != -1:
            read_file_processor(log, global_data)
        elif message.find("fuse_debug") != -1 and message.find("<-") != -1:
            kernel_call_processor(log, global_data)
        elif message.find("fuse_debug") != -1 and message.find("->") != -1:
            response_processor(log, global_data)

    # not_returned_request_processor(global_data)

    return global_data

# Processor.gen_processor = gen_processor


