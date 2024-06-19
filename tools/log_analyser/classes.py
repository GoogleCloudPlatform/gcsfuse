class Calls:
    def __init__(self, name):
        self.call_name = name
        self.calls_made = 0
        self.calls_returned = 0
        self.response_times = []
        self.total_response_time = 0
        self.not_returned_calls = []


class GlobalCalls:
    def __init__(self):
        self.calls = [Calls("unlink"),
                      Calls("rename"),
                      Calls("mkdir"),
                      Calls("releasedirhandle"),
                      Calls("createfile"),
                      Calls("rmdir")]


class GlobalData:
    bytes_from_gcs = 0
    bytes_to_gcs = 0
    requests = {}
    name_object_map = {}
    inode_name_map = {}
    handle_name_map = {}
    kernel_calls = GlobalCalls()


class Handle:
    def __init__(self, num, time_sec, time_nano):
        self.handle_num = num
        self.opening_time = time_sec
        self.opening_time_nano = time_nano
        self.closing_time = 0
        self.closing_time_nano = 0
        self.read_pattern = ""
        self.total_reads = 0
        self.total_writes = 0
        self.total_read_size = 0
        self.total_write_size = 0
        self.last_read_offset = -1
        self.read_times = []
        self.write_times = []


class GcsCalls:
    def __init__(self):
        self.calls = [Calls("StatObject"),
                      Calls("ListObjects"),
                      Calls("CopyObject"),
                      Calls("ComposeObjects"),
                      Calls("UpdateObject"),
                      Calls("DeleteObject"),
                      Calls("CreateObject"),
                      Calls("Read")]


class Object:
    def __init__(self, inode, parent, rel_name, abs_name):
        self.inode = inode
        self.parent = parent
        self.rel_name = rel_name
        self.abs_name = abs_name
        self.gcs_calls = GcsCalls()
        self.is_dir = False
        self.handles = {}
        self.kernel_calls = KernelCalls()
        self.opened_handles = 0
        self.open_tup = []
        self.closed_handles = 0
        self.close_tup = []


class KernelCalls:
    def __init__(self):
        self.calls = [Calls("lookup"),
                      Calls("readfile"),
                      Calls("openfile"),
                      Calls("flushfile"),
                      Calls("writefile"),
                      Calls("createsymlink"),
                      Calls("readsymlink"),
                      Calls("releasefilehandle"),
                      Calls("opendir"),
                      Calls("readdir")]


class Request:
    def __init__(self, typ, obj):
        self.req_type = typ
        self.object_name = obj
        self.inode = 0
        self.handle = 0
        self.parent = 0
        self.rel_name = None
        self.timestamp_sec = 0
        self.timestamp_nano = 0
        self.keyword = ""

