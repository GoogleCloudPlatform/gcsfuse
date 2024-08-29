from parser.calls import Calls

class GcsCalls:
    """GCS call info specific to a file/dir"""
    def __init__(self):
        self.calls = [Calls("StatObject"),
                      Calls("ListObjects"),
                      Calls("CopyObject"),
                      Calls("ComposeObjects"),
                      Calls("UpdateObject"),
                      Calls("DeleteObject"),
                      Calls("CreateObject"),
                      Calls("Read"),
                      Calls("RenameFolder"),
                      Calls("CreateFolder"),
                      Calls("DeleteFolder"),
                      Calls("GetFolder")]
        self.callname_index_map = {"StatObject": 0, "ListObjects": 1,
                                   "CopyObject": 2, "ComposeObjects": 3,
                                   "UpdateObject": 4, "DeleteObject": 5,
                                   "CreateObject": 6, "Read": 7,
                                   "RenameFolder": 8, "CreateFolder": 9,
                                   "DeleteFolder": 10, "GetFolder": 11}