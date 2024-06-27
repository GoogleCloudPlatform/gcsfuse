from parser.calls import Calls

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
        self.callname_index_map = {"StatObject": 0, "ListObjects": 1, "CopyObject": 2, "ComposeObjects": 3, "UpdateObject": 4, "DeleteObject": 5, "CreateObject": 6, "Read": 7}