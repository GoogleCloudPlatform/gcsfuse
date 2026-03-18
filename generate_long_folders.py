import os
import random
import string

def generate_folders(base_path, num_folders):
    # Maximum file/folder name length in most file systems is 255 bytes.
    # We use 255 characters to maximize the discrepancy between the
    # pointer size (8 bytes) and the actual memory used by the Folder
    # struct + string content in GCSFuse's stat_cache.
    name_length = 255

    # Create the base directory if it doesn't exist
    if not os.path.exists(base_path):
        os.makedirs(base_path)

    print(f"Generating {num_folders} folders with {name_length}-character names in '{base_path}'...")

    for i in range(num_folders):
        # Generate a random string of length 255
        random_name = ''.join(random.choices(string.ascii_letters + string.digits, k=name_length))

        folder_path = os.path.join(base_path, random_name)

        try:
            os.makedirs(folder_path)
        except OSError as e:
            print(f"Error creating folder {folder_path}: {e}")
            break

        if (i + 1) % 1000 == 0:
            print(f"Generated {i + 1} folders...")

    print("Done!")

if __name__ == "__main__":
    # Generate 100,000 folders.
    # In GCSFuse's stat cache, if the size of each folder name is 255 bytes,
    # the actual heap memory consumed per folder entry would be at least
    # 255 bytes (name) + 40 bytes (Folder struct) = 295 bytes.
    # However, because of the bug `util.UnsafeSizeOf(&e.f)`, GCSFuse
    # calculates the size of the folder as just 8 bytes (the pointer size).
    # This 100,000 folder test would consume ~29.5 MB of actual heap memory
    # for the folder structs, but the cache will only account for ~0.8 MB.
    # This script can be run against a GCSFuse mount to prove the discrepancy.
    generate_folders("./test_long_folders", 100000)
