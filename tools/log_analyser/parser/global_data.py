from parser.global_calls import GlobalCalls


class GlobalData:
    """
    contains all the dictionaries to store all the file/dir objects, mappings from
    inode number to file names, handles to file names, also stores faulty logs list
    and list of requests present in the logs, pops when the response log comes
    """
    bytes_from_gcs = 0
    bytes_to_gcs = 0
    requests = {}
    name_object_map = {}
    inode_name_map = {}
    handle_name_map = {}
    gcalls = GlobalCalls()
    max_read_entries = []
    max_listobjects_entries = []
    max_createobject_entries = []
    faulty_logs = []
