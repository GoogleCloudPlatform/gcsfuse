import json
import decimal
import sys
sys.path.append('/home/ashmeen_google_com/github/gcsfuse/perfmetrics/scripts')
from gsheet import gsheet

FLUSH_WORKSHEET_NAME="flush_write_time"

def get_times_from_log(log_file):
    """
    Extracts time duration of "CreateFile", "FlushFile", and its corresponding "OK"
     response from a log file, assuming only single file write.

    Args:
        log_file (str): Path to the log file.

    Returns:
        dict: A dictionary with time duration for "WriteFile" and "FlushFile".
    """

    flush_opcode = None
    create_time = None
    flush_time = None
    flush_ok_time = None

    with open(log_file, 'r') as f:
        for line in f:
            try:
                log_entry = json.loads(line)
                message = log_entry["message"]
                timestamp_seconds = log_entry["timestamp"]["seconds"]
                timestamp_nanos = log_entry["timestamp"]["nanos"]
                timestamp = f"{timestamp_seconds}.{timestamp_nanos:09d}"

                if "CreateFile" in message:
                    create_time = timestamp

                elif "FlushFile" in message:
                    flush_time = timestamp
                    flush_opcode = message.split("Op")[1].split()[0]  # Extract opcode

                elif flush_opcode is not None and flush_opcode in message:
                    flush_ok_time = timestamp

            except json.JSONDecodeError as e:
                print(f"Warning: Skipping invalid JSON line: {line.strip()} - Error: {e}")

    create_time_decimal = decimal.Decimal(create_time)  # Convert to Decimal
    flush_time_decimal = decimal.Decimal(flush_time)
    flush_ok_time_decimal = decimal.Decimal(flush_ok_time)
    times = {
            "WriteFile": str(flush_time_decimal - create_time_decimal),
            "FlushFile": str(flush_ok_time_decimal - flush_time_decimal),
        }
    return times

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python script.py <log_file_path> <spreadsheet_id>")
        sys.exit(1)

    log_file_path = sys.argv[1]
    spreadsheet_id = sys.argv[2]

    timestamps = get_times_from_log(log_file_path)
    data = [
            (log_file_path, timestamps["WriteFile"], timestamps["FlushFile"])
    ]
    gsheet.append_to_google_sheet(FLUSH_WORKSHEET_NAME, data, spreadsheet_id)
