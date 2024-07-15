from parser.calls import Calls

class GlobalCalls:
    """
    here all the call info is stored globally
    """
    def __init__(self):
        self.kernel_calls = [Calls("Unlink"),
                             Calls("Rename"),
                             Calls("MkDir"),
                             Calls("ReleaseDirHandle"),
                             Calls("CreateFile"),
                             Calls("RmDir"),
                             Calls("LookUpInode"),
                             Calls("ReadFile"),
                             Calls("OpenFile"),
                             Calls("FlushFile"),
                             Calls("WriteFile"),
                             Calls("CreateSymLink"),
                             Calls("ReadSymLink"),
                             Calls("ReleaseFileHandle"),
                             Calls("OpenDir"),
                             Calls("ReadDir")]
        self.gcs_calls = [Calls("StatObject"),
                          Calls("ListObjects"),
                          Calls("CopyObject"),
                          Calls("ComposeObjects"),
                          Calls("UpdateObject"),
                          Calls("DeleteObject"),
                          Calls("CreateObject"),
                          Calls("Read")]
        self.kernel_index_map = {"Unlink": 0, "Rename": 1, "MkDir": 2, "ReleaseDirHandle": 3, "CreateFile": 4, "RmDir": 5, "LookUpInode": 6, "ReadFile": 7, "OpenFile": 8, "FlushFile": 9, "WriteFile": 10, "CreateSymLink": 11, "ReadSymLink": 12, "ReleaseFileHandle": 13, "OpenDir": 14, "ReadDir": 15}
        self.gcs_index_map = {"StatObject": 0, "ListObjects": 1, "CopyObject": 2, "ComposeObjects": 3, "UpdateObject": 4, "DeleteObject": 5, "CreateObject": 6, "Read": 7}