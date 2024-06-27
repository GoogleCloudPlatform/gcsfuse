from parser.calls import Calls

class KernelCalls:
    def __init__(self):
        self.calls = [Calls("LookUpInode"),
                      Calls("ReadFile"),
                      Calls("OpenFile"),
                      Calls("FlushFile"),
                      Calls("WriteFile"),
                      Calls("CreateSymLink"),
                      Calls("ReadSymLink"),
                      Calls("ReleaseFileHandle"),
                      Calls("OpenDir"),
                      Calls("ReadDir")]
        self.callname_index_map = {"LookUpInode": 0, "ReadFile": 1, "OpenFile": 2, "FlushFile": 3, "WriteFile": 4, "CreateSymLink": 5, "ReadSymLink": 6, "ReleaseFileHandle": 7, "OpenDir": 8, "ReadDir": 9}