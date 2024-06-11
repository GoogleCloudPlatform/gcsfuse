
def get_val(message, key, delim, direction, offset):
    # offset contains adjustments needed for spaces and key lengths
    if direction == "fwd":
        start_index = message.find(key)+len(key)+offset
    else:
        start_index = message.rfind(key)+len(key)+offset
    end_index = message.find(delim, start_index)
    return message[start_index:end_index]


def give_dir_tag(global_data, inode):
    name = global_data.inode_name_map[inode]
    global_data.name_object_map[name].is_dir = True








