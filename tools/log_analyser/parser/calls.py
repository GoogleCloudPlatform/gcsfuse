class Calls:
    def __init__(self, name):
        self.call_name = name
        self.calls_made = 0
        self.calls_returned = 0
        self.response_times = []
        self.total_response_time = 0
        self.not_returned_calls = []