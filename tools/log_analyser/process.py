import utility
from classes import Object as Object
from classes import Handle as Handle
from classes import GlobalData as GlobalData
from classes import GcsCalls as GcsCalls
from classes import Calls as Calls
from classes import Request as Request


def lookup_processor(log, global_data):
    message = log["message"]
    req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
    name = utility.get_val(message, "name", "\"", "fwd", 2)
    parent = int(utility.get_val(message, "parent", ",", "fwd", 1))
    if parent != 0 and parent != 1:
        # give_dir_tag(global_data, parent)
        prefix_name = global_data.inode_name_map[parent]
        prefix_name += "/"
    else:
        prefix_name = ""
    abs_name = prefix_name + name
    global_data.requests[req_id] = Request("lookup", abs_name)
    global_data.requests[req_id].parent = parent
    global_data.requests[req_id].rel_name = name
    # global_data.requests[req_id].timestamp_sec = log["timestamp"]["seconds"]
    # global_data.requests[req_id].timestamp_nano = log["timestamp"]["nanos"]


def response_processor(log, global_data):
    message = log["message"]
    time_sec = log["timestamp"]["seconds"]
    time_nano = log["timestamp"]["nanos"]
    req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
    if req_id in global_data.requests.keys():
        req = global_data.requests[req_id]
        if req.object_name in global_data.name_object_map.keys():
            obj = global_data.name_object_map[req.object_name]
        else:
            global_data.name_object_map[req.object_name] = Object(0, 0, "", "")
            obj = global_data.name_object_map[req.object_name]

        if req.req_type == "lookup":
            if message.find("Error") != -1:
                if req.object_name in global_data.name_object_map.keys():
                    obj.is_dir = False
                    global_data.requests.pop(req_id)
                    return
            inode = int(utility.get_val(message, "(inode", ")", "fwd", 1))
            obj.inode = inode
            obj.parent = req.parent
            obj.rel_name = req.rel_name
            obj.abs_name = req.object_name
            obj.kernel_calls.calls[0].calls_made += 1
            global_data.inode_name_map[inode] = req.object_name

        elif req.req_type == "openfile":
            handle = int(utility.get_val(message, "handle", ")", "fwd", 1))
            obj.handles[handle] = Handle(handle, req.timestamp_sec, req.timestamp_nano)
            obj.opened_handles += 1
            obj.open_tup.append([[int(req.timestamp_sec), int(req.timestamp_nano)], obj.opened_handles])
            obj.kernel_calls.calls[2].calls_made += 1
            global_data.handle_name_map[handle] = req.object_name

        elif req.req_type == "createfile":
            inode = int(utility.get_val(message, "inode", ")", "fwd", 1))
            obj.inode = inode
            obj.parent = req.parent
            obj.abs_name = req.object_name
            obj.rel_name = req.rel_name
            global_data.kernel_calls.calls[4].calls_made += 1

        elif req.req_type == "readfile":
            obj.handles[req.handle].read_times.append(time_sec + 1e-9*time_nano - req.timestamp_sec - 1e-9*req.timestamp_nano)

        elif req.req_type == "writefile":
            obj.handles[req.handle].write_times.append(time_sec + 1e-9*time_nano - req.timestamp_sec - 1e-9*req.timestamp_nano)

        global_data.requests.pop(req_id)


def gcs_call_processor(log, global_data):
    message = log["message"]
    name = utility.get_val(message, "(", "\"", "fwd", 1)
    if name not in global_data.name_object_map.keys():
        global_data.name_object_map[name] = Object(None, None, None, name)
    if len(name) > 0 and name[len(name)-1] == "/":
        global_data.name_object_map[name].is_dir = True

    if message.find("StatObject") != -1:
        # call_obj = global_data.name_object_map[name].gcs_calls.statobject_calls
        call_obj = global_data.name_object_map[name].gcs_calls.calls[0]
        if message.find("gcs.NotFoundError: storage: object doesn't exist") != -1:
            global_data.name_object_map[name].is_dir = True

    elif message.find("ListObjects") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[1]

    elif message.find("CopyObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[2]

    elif message.find("ComposeObjects") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[3]

    elif message.find("UpdateObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[4]

    elif message.find("DeleteObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[5]

    elif message.find("CreateObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[6]

    elif message.find("Read") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.calls[7]
        if message.find("->") != -1 and message.find("nil") == -1:
            start = int(utility.get_val(message, "[", ",", "fwd", 0))
            final = int(utility.get_val(message, ",", ")", "bck", 1))
            global_data.bytes_from_gcs += final-start

    if message.find("<-") != -1:
        req = utility.get_val(message, "0x", " ", "fwd", 0)
        global_data.requests[req] = Request("gcsreq", name)
        global_data.requests[req].timestamp_sec = log["timestamp"]["seconds"]
        global_data.requests[req].keyword = call_obj.call_name
        call_obj.calls_made += 1

    elif message.find("->") != -1:
        req = utility.get_val(message, "0x", " ", "fwd", 0)

        if req in global_data.requests.keys():
            call_obj.calls_returned += 1
            start_index = message.rfind("(")

            if message.find("ms", start_index) != -1:
                call_obj.total_response_time += float(utility.get_val(message, "(", "ms", "bck", 0))
                call_obj.response_times.append(float(utility.get_val(message, "(", "ms", "bck", 0)))
            else:
                call_obj.total_response_time += 1000*float(utility.get_val(message, "(", "s", "bck", 0))
                call_obj.response_times.append(1000*float(utility.get_val(message, "(", "s", "bck", 0)))

            global_data.requests.pop(req)


def open_file_processor(log, global_data):
    message = log["message"]
    inode = int(utility.get_val(message, "inode", ",", "fwd", 1))
    req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
    global_data.requests[req_id] = Request("openfile", global_data.inode_name_map[inode])
    global_data.requests[req_id].inode = inode
    global_data.requests[req_id].timestamp_sec = log["timestamp"]["seconds"]
    global_data.requests[req_id].timestamp_nano = log["timestamp"]["nanos"]


def release_file_handle_processor(log, global_data):
    message = log["message"]
    handle = int(utility.get_val(message, ", handle", ")", "fwd", 1))
    if handle in global_data.handle_name_map.keys():
        obj = global_data.name_object_map[global_data.handle_name_map[handle]]
        obj.closed_handles += 1
        obj.handles[handle].closing_time = log["timestamp"]["seconds"]
        obj.handles[handle].closing_time_nano = log["timestamp"]["nanos"]
        obj.close_tup.append([[int(log["timestamp"]["seconds"]), int(log["timestamp"]["nanos"])], obj.closed_handles])
        obj.kernel_calls.calls[7].calls_made += 1


def read_file_processor(log, global_data):
    message = log["message"]
    inode = int(utility.get_val(message, "inode", ",", "fwd", 1))
    handle = int(utility.get_val(message, "handle", ",", "fwd", 1))
    offset = int(utility.get_val(message, "offset", ",", "fwd", 1))
    byts = int(utility.get_val(message, ",", " ", "bck", 1))
    req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
    obj = global_data.name_object_map[global_data.inode_name_map[inode]]
    handle_obj = obj.handles[handle]
    if message.find("ReadFile") != -1:
        global_data.requests[req_id] = Request("readfile", global_data.inode_name_map[inode])
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
        global_data.requests[req_id] = Request("writefile", global_data.inode_name_map[inode])
        handle_obj.total_writes += 1
        handle_obj.total_write_size += byts
        global_data.bytes_to_gcs += byts
        obj.kernel_calls.calls[4].calls_made += 1
    global_data.requests[req_id].inode = inode
    global_data.requests[req_id].handle = handle
    global_data.requests[req_id].timestamp_sec = log["timestamp"]["seconds"]
    global_data.requests[req_id].timestamp_nano = log["timestamp"]["nanos"]

def kernel_call_processor(log, global_data):
    message = log["message"]
    inode = -1
    name = None
    if message.find("inode") != -1:
        inode = int(utility.get_val(message, "inode", ",", "fwd", 1))
    elif message.find("name") != -1:
        name = utility.get_val(message, "name", "\"", "fwd", 2)

    if message.find("FlushFile") != -1:
        global_data.name_object_map[global_data.inode_name_map[inode]].kernel_calls.calls[3].calls_made += 1
    elif message.find("ReadDir") != -1:
        global_data.name_object_map[global_data.inode_name_map[inode]].kernel_calls.calls[9].calls_made += 1
    elif message.find("OpenDir") != -1:
        global_data.name_object_map[global_data.inode_name_map[inode]].kernel_calls.calls[8].calls_made += 1
    elif message.find("ReadSymLink") != -1:
        global_data.name_object_map[global_data.inode_name_map[inode]].kernel_calls.calls[6].calls_made += 1
    elif message.find("CreateSymLink") != -1:
        global_data.name_object_map[name].kernel_calls.calls[5].calls_made += 1
    elif message.find("Unlink") != -1:
        global_data.kernel_calls.calls[0].calls_made += 1
    elif message.find("Rename") != -1:
        global_data.kernel_calls.calls[1].calls_made += 1
    elif message.find("RmDir") != -1:
        global_data.kernel_calls.calls[5].calls_made += 1
    elif message.find("MkDir") != -1:
        global_data.kernel_calls.calls[2].calls_made += 1
    elif message.find("ReleaseDirHandle") != -1:
        global_data.kernel_calls.calls[3].calls_made += 1
    elif message.find("CreateFile") != -1:
        name = utility.get_val(message, "name", "\"", "fwd", 2)
        parent = int(utility.get_val(message, "parent", ",", "fwd", 1))
        prefix = global_data.inode_name_map[parent] + "/"
        req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
        global_data.requests[req_id] = Request("createfile", prefix + name)
        global_data.requests[req_id].rel_name = name
        global_data.requests[req_id].parent = parent
        global_data.calls.createfile += 1


def not_returned_request_processor(global_data):
    for req in global_data.requests:
        if global_data.requests[req].req_type == "gcsreq":
            obj = global_data.name_object_map[req.object_name]
            if req.keyword == "StatObject":
                obj.gcs_calls.calls[0].not_returned.append(req.timestamp_sec)
            elif req.keyword == "ListObjects":
                obj.gcs_calls.calls[1].not_returned.append(req.timestamp_sec)
            elif req.keyword == "UpdateObject":
                obj.gcs_calls.calls[4].not_returned.append(req.timestamp_sec)
            elif req.keyword == "CreateObject":
                obj.gcs_calls.calls[6].not_returned.append(req.timestamp_sec)
            elif req.keyword == "DeleteObject":
                obj.gcs_calls.calls[5].not_returned.append(req.timestamp_sec)
            elif req.keyword == "CopyObject":
                obj.gcs_calls.calls[2].not_returned.append(req.timestamp_sec)
            elif req.keyword == "Read":
                obj.gcs_calls.calls[7].not_returned.append(req.timestamp_sec)
            elif req.keyword == "ComposeObjects":
                obj.gcs_calls.calls[3].not_returned.append(req.timestamp_sec)

        # elif global_data.requests[req].req_type == "readfile":



# class Processor:
#     lookup_processor = lookup_processor
#     response_processor = response_processor
#     gcs_call_processor = gcs_call_processor
#     open_file_processor = open_file_processor
#     release_file_handle_processor = release_file_handle_processor
#     read_file_processor = read_file_processor


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
        elif message.find("->") != -1:
            response_processor(log, global_data)

    not_returned_request_processor(global_data)

    return global_data

# Processor.gen_processor = gen_processor


