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


def response_processor(log, global_data):
    message = log["message"]
    req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
    if req_id in global_data.requests.keys():
        req = global_data.requests[req_id]
        if req.object_name in global_data.name_object_map.keys():
            obj = global_data.name_object_map[req.object_name]
        if req.req_type == "lookup":
            if message.find("Error") != -1:
                if req.object_name in global_data.name_object_map.keys():
                    obj.is_dir = False
                    return
            inode = int(utility.get_val(message, "(inode", ")", "fwd", 1))
            obj.inode = inode
            obj.parent = req.parent
            obj.rel_name = req.rel_name
            obj.abs_name = req.object_name
            global_data.inode_name_map[inode] = req.object_name

        if req.req_type == "openfile":
            handle = int(utility.get_val(message, "handle", ")", "fwd", 1))
            obj.handles[handle] = Handle(handle, req.timestamp_sec)
            obj.opened_handles += 1
            obj.open_tup.append([req.timestamp_sec, obj.opened_handles])
            global_data.handle_name_map[handle] = req.object_name

        global_data.requests.pop(req_id)


def gcs_call_processor(log, global_data):
    message = log["message"]
    name = utility.get_val(message, "(", "\"", "fwd", 1)
    if name not in global_data.name_object_map.keys():
        global_data.name_object_map[name] = Object(None, None, None, name)
    if name[len(name)-1] == "/":
        global_data.name_object_map[name].is_dir = True

    if message.find("StatObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.statobject_calls
        if message.find("gcs.NotFoundError: storage: object doesn't exist") != -1:
            global_data.name_object_map[name].is_dir = True

    elif message.find("ListObjects") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.listobjects_calls

    elif message.find("ComposeObjects") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.composeobjects_calls

    elif message.find("CreateObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.createobject_calls

    elif message.find("UpdateObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.updateobject_calls

    elif message.find("DeleteObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.deleteobject_calls

    elif message.find("CopyObject") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.copyobject_calls

    elif message.find("Read") != -1:
        call_obj = global_data.name_object_map[name].gcs_calls.read_calls
        if message.find("->") != -1 and message.find("nil") == -1:
            start = int(utility.get_val(message, "[", ",", "fwd", 0))
            final = int(utility.get_val(message, ",", ")", "bck", 1))
            global_data.bytes_from_gcs += final-start

    if message.find("<-") != -1:
        req = utility.get_val(message, "0x", " ", "fwd", 0)
        global_data.requests[req] = Request("gcsreq", name)
        global_data.requests[req].timestamp_sec = log["timestamp"]["seconds"]
        call_obj.calls_made += 1

    elif message.find("->") != -1:
        req = utility.get_val(message, "0x", " ", "fwd", 0)

        if req in global_data.requests.keys():
            call_obj.calls_returned += 1
            start_index = message.rfind("(")

            if message.find("ms", start_index) != -1:
                call_obj.total_response_time += float(utility.get_val(message, "(", "ms", "bck", 0))
            else:
                call_obj.total_response_time += 1000*float(utility.get_val(message, "(", "s", "bck", 0))

            global_data.requests.pop(req)


def open_file_processor(log, global_data):
    message = log["message"]
    inode = int(utility.get_val(message, "inode", ",", "fwd", 1))
    req_id = utility.get_val(message, "Op 0x", " ", "fwd", 0)
    global_data.requests[req_id] = Request("openfile", global_data.inode_name_map[inode])
    global_data.requests[req_id].inode = inode
    global_data.requests[req_id].timestamp_sec = log["timestamp"]["seconds"]


def release_file_handle_processor(log, global_data):
    message = log["message"]
    handle = int(utility.get_val(message, ", handle", ")", "fwd", 1))
    if handle in global_data.handle_name_map.keys():
        obj = global_data.name_object_map[global_data.handle_name_map[handle]]
        obj.closed_handles += 1
        obj.handles[handle].closing_time = log["timestamp"]["seconds"]
        obj.close_tup.append([log["timestamp"]["seconds"], obj.closed_handles])


def read_file_processor(log, global_data):
    message = log["message"]
    inode = int(utility.get_val(message, "inode", ",", "fwd", 1))
    handle = int(utility.get_val(message, "handle", ",", "fwd", 1))
    offset = int(utility.get_val(message, "offset", ",", "fwd", 1))
    byts = int(utility.get_val(message, ",", " ", "bck", 1))
    obj = global_data.name_object_map[global_data.inode_name_map[inode]]
    handle_obj = obj.handles[handle]
    if handle_obj.last_read_offset == -1:
        handle_obj.read_pattern += "_"
    elif handle_obj.last_read_offset == offset:
        handle_obj.read_pattern += "s"
    else:
        handle_obj.read_pattern += "r"
    handle_obj.last_read_offset = offset + byts
    handle_obj.total_read_size += byts
    handle_obj.total_reads += 1

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
        elif message.find("ReadFile") != -1:
            read_file_processor(log, global_data)
        elif message.find("->") != -1:
            response_processor(log, global_data)

    for itr in global_data.name_object_map.keys():
        print(itr, ":", global_data.name_object_map[itr].inode, global_data.name_object_map[itr].gcs_calls.listobjects_calls.calls_made, global_data.name_object_map[itr].is_dir)
        print("Handles:")
        obj = global_data.name_object_map[itr]
        for handle in obj.handles.keys():
            handle_obj = obj.handles[handle]
            print("Number:", handle_obj.handle_num, "total reads:", handle_obj.total_reads, "pattern:", handle_obj.read_pattern)
            print("opening time:", handle_obj.opening_time, "closing time:", handle_obj.closing_time)

        print("")



# Processor.gen_processor = gen_processor


