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
        self.open_pos = 0
        self.close_pos = 0