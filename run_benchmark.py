
import subprocess
import time
import os
import signal
import re
import csv

# --- Configuration ---
GCSFUSE_CONFIG_FILE = os.path.expanduser("~/tmp/gcsfuse_config_file_no_tracing_profiling.yaml")
FIO_JOB_FILE = os.path.expanduser("~/tmp/gcsfuse_sample_job_spec_randread.fio")
BUCKET_NAME = "test_thrivikramks_1"
MOUNT_POINT = os.path.expanduser("~/tmp/mnt")
FIO_TEST_DIR = os.path.join(MOUNT_POINT, "fio_tests")
# FIO_OUTPUT_FILE = os.path.expanduser("~/tmp/fio_randreadoutput.json")
SCRIPT_OUTPUT_FILE = os.path.expanduser("~/tmp/benchmark_results.csv")
GCSFUSE_WORKSPACE = os.path.expanduser("~/workspace/gcsfuse")

# Block sizes in MB
BLOCK_SIZES_MB = [1] + list(range(10, 151, 10))
# BLOCK_SIZES_MB = [1] + list(range(10, 11, 10))

DATA_KEYS = ["block_size", "read_bw", "fio_output"]
CSV_HEADERS = ["Block Size", "Read BW", "Complete FIO Output"]
fieldnames = DATA_KEYS

def update_config_file(file_path, pattern, replacement):
    """Updates a file by replacing a pattern."""
    try:
        with open(file_path, 'r') as f:
            content = f.read()
        
        content = re.sub(pattern, replacement, content)

        with open(file_path, 'w') as f:
            f.write(content)
    except FileNotFoundError:
        print(f"Error: Config file not found at {file_path}")
        exit(1)
    except Exception as e:
        print(f"Error updating config file {file_path}: {e}")
        exit(1)

def run_benchmark():
    """Runs the gcsfuse and fio benchmark."""

    # Ensure mount point and test directory exist
    os.makedirs(FIO_TEST_DIR, exist_ok=True)

    fio_results = []

    for size_mb in BLOCK_SIZES_MB:
        print(f"--- Running test for block size: {size_mb}MB ---")

        # 1. Update gcsfuse config
        block_size_str = f"{size_mb}MB"
        update_config_file(
            GCSFUSE_CONFIG_FILE,
            r"(label: .*_)\d+MB(_.*)",
            fr"\g<1>{block_size_str}\g<2>"
        )

        # 2. Update fio config
        update_config_file(
            FIO_JOB_FILE,
            r"(bs=)\d+M",
            fr"\g<1>{size_mb}M"
        )

        # 3. Start gcsfuse
        gcsfuse_command = [
            "go", "run", ".",
            "--implicit-dirs",
            "--config-file", GCSFUSE_CONFIG_FILE,
            BUCKET_NAME,
            MOUNT_POINT
        ]
        
        print(f"Starting gcsfuse: {' '.join(gcsfuse_command)}")
        gcsfuse_process = subprocess.Popen(gcsfuse_command, cwd=GCSFUSE_WORKSPACE, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, preexec_fn=os.setsid)
        
        # Give gcsfuse a moment to mount
        time.sleep(5)

        # 4. Run fio
        fio_command = ["fio", FIO_JOB_FILE]
        # Can also add options: "--output-format=json", "--output=fio_randreadoutput.json"
        print(f"Running fio: {' '.join(fio_command)}")
        
        try:
            fio_output = subprocess.check_output(fio_command, stderr=subprocess.STDOUT, text=True)

            print("--- FIO Output ---")
            print(fio_output)
            match = re.search(r"BW=(\d+)MiB/s", fio_output)
            bw_value = None
            read_line = None
            if match:
                bw_value = int(match.group(1))
                print(f"Bandwidth: {bw_value} MiB/s")
            read_line_match = re.search(r"READ:.*", fio_output)
            if read_line_match:
                read_line = read_line_match.group(0)
            
            print("--- End FIO Output ---")
    
            fio_result = {
                "block_size": f'{size_mb}MB',
                "read_bw": f'{bw_value}MiB/s',
                "fio_output": read_line,
            }

            fio_results.append(fio_result)

        except subprocess.CalledProcessError as e:
            print(f"Fio command failed with exit code {e.returncode}")
            print("Fio output:")
            print(e.output)
        
        finally:
            # 5. Stop gcsfuse gracefully
            print("Stopping gcsfuse...")
            gcsfuse_process.send_signal(signal.SIGINT)
            try:
                gcsfuse_process.wait(timeout=120)
                print("gcsfuse stopped.")
            except subprocess.TimeoutExpired:
                print("gcsfuse did not stop gracefully, killing.")
                gcsfuse_process.kill()
            
            # Clean up mount point
            subprocess.run(["fusermount", "-u", MOUNT_POINT])
            time.sleep(2) # Give it a moment to unmount

    with open(SCRIPT_OUTPUT_FILE, 'w', newline='') as csvfile:
        csv.writer(csvfile).writerow(CSV_HEADERS)

        writer = csv.DictWriter(csvfile, fieldnames=fieldnames, extrasaction='ignore')

        print('fio results')
        print(fio_results)
        writer.writerows(fio_results)

if __name__ == "__main__":
    run_benchmark()
