class GlobalData:
    bytes_from_gcs = 0
    bytes_to_gcs = 0
    requests = {}
    name_object_map = {}
    inode_name_map = {}
    handle_name_map = {}

class Handle:
    def __init__(self, num, time_sec):
        self.handle_num = num
        self.opening_time = time_sec
        self.closing_time = 0
        self.read_pattern = ""
        self.total_reads = 0
        self.total_read_size = 0
        self.last_read_offset = -1


class Calls:
    def __init__(self, name):
        self.call_name = name
        self.calls_made = 0
        self.calls_returned = 0
        self.total_response_time = 0
        self.not_returned_calls = []


class GcsCalls:
    def __init__(self):
        self.statobject_calls = Calls("StatObject")
        self.listobjects_calls = Calls("ListObjects")
        self.copyobject_calls = Calls("CopyObject")
        self.composeobjects_calls = Calls("ComposeObjects")
        self.updateobject_calls = Calls("UpdateObject")
        self.deleteobject_calls = Calls("DeleteObject")
        self.createobject_calls = Calls("CreateObject")
        self.read_calls = Calls("Read")


class Object:
    def __init__(self, inode, parent, rel_name, abs_name):
        self.inode = inode
        self.parent = parent
        self.rel_name = rel_name
        self.abs_name = abs_name
        self.gcs_calls = GcsCalls()
        self.is_dir = False
        self.handles = {}
        self.kernel_calls = None
        self.opened_handles = 0
        self.open_tup = []
        self.closed_handles = 0
        self.close_tup = []


class KernelCalls:
    lookupinode_calls = None


class Request:
    def __init__(self, typ, obj):
        self.req_type = typ
        self.object_name = obj
        self.inode = 0
        self.handle = 0
        self.parent = 0
        self.rel_name = None
        self.timestamp_sec = 0

