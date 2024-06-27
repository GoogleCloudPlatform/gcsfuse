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
        self.is_valid = True
        self.is_lookup_call_added = False