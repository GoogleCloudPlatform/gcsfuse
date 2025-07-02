import os


def read_with_odirect(filepath, buffer_size):
    """
    Reads a file using O_DIRECT flag.

    Args:
        filepath (str): The path to the file.
        buffer_size (int): The size of the buffer to use for reading.
                           Must be a multiple of the block size of the underlying device.

    Returns:
        bytes: The content of the file.
    """
    try:
        fd = os.open(filepath, os.O_RDONLY | os.O_DIRECT)
        content = 0
        while True:
            # Read in chunks
            chunk = os.read(fd, buffer_size)
            if not chunk:
                break
            content += len(chunk)
        os.close(fd)
        return content
    except OSError as e:
        print(f"Error reading file with O_DIRECT: {e}")
        print(
            "Note: O_DIRECT might not be supported on this platform/filesystem or requires root privileges."
        )
        return None


if __name__ == "__main__":
    file_to_read = "/home/princer_google_com/bucket/data10G.txt"
    read_buffer_size = 1024 * 1024  # 1MB
    print("Reading file with O_DIRECT flag")
    size = read_with_odirect(file_to_read, read_buffer_size)
    print("File size:", size, "bytes", "Successfully read.")
