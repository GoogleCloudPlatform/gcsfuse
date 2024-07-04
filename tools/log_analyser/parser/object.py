from parser.gcs_calls import GcsCalls
from parser.kernel_calls import KernelCalls

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
        self.read_pattern = ""
        self.read_bytes = []
        self.read_ranges = []
        self.last_read_offset = -1