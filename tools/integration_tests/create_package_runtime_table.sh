#!/usr/bin/env python3
# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
This script generates a table of test package runtimes using the 'rich' library.

Usage:
    python3 create_package_runtime_table.sh <FILE_PATH>

Requirements:
    Requires 'rich' library installed on the system (e.g., 'pip install rich').

Input File Format (<FILE_PATH>):
    Space-separated lines with the following fields:
    <package_name> <bucket_type> <exit_code> <start_time_seconds> <end_time_seconds> [<attempt>]

Example File Content:
    pkg_name bucket_type exit_code start_time_seconds end_time_seconds attempt
    pkg1 bucket-standard 0 0 120 0
    pkg2 bucket-premium 0 60 180 1
    pkg3 bucket-standard 1 120 240 2
"""
import sys, os

# Column indices for the input file data
IDX_PKG_NAME = 0
IDX_BUCKET_TYPE = 1
IDX_EXIT_CODE = 2
IDX_START_TIME = 3
IDX_END_TIME = 4
IDX_ATTEMPT = 5

MIN_REQUIRED_FIELDS = 5

# Minimum widths based on header lengths
MIN_LEN_PKG_NAME_HEADER = 12
MIN_LEN_BUCKET_TYPE_HEADER = 11
MIN_LEN_RUNTIME_BAR_HEADER = 31

# Estimated padding for table columns
PADDING_TIME_COL = 8
PADDING_STATUS_COL = 25 # Increased to fit '✅ FLAKY (Attempt X)'
PADDING_BORDERS = 20

WIDTH_FALLBACK = 80
SECONDS_PER_MINUTE = 60

# Verify command line arguments
if len(sys.argv) != 2:
    print(f"Usage: {sys.argv[0]} <FILE_PATH>")
    sys.exit(1)

path = sys.argv[1]
if not os.path.isfile(path):
    print(f"Error: File '{path}' not found.")
    sys.exit(1)

# Read input file, filter out empty lines, and sort alphabetically
with open(path) as f:
    lines = sorted([l.split() for l in f if l.strip()])

# Use the 'rich' library to generate a pretty table visualization
try:
    from rich.console import Console
    from rich.table import Table
    import shutil

    # Calculate optimal table width based on content
    valid_lines = [p for p in lines if len(p) >= MIN_REQUIRED_FIELDS]
    if valid_lines:
        max_pkg = max(MIN_LEN_PKG_NAME_HEADER, max(len(p[IDX_PKG_NAME]) for p in valid_lines))
        max_type = max(MIN_LEN_BUCKET_TYPE_HEADER, max(len(p[IDX_BUCKET_TYPE]) for p in valid_lines))
        max_rt = max(MIN_LEN_RUNTIME_BAR_HEADER, max(int(p[IDX_START_TIME]) // SECONDS_PER_MINUTE + (int(p[IDX_END_TIME]) - int(p[IDX_START_TIME]) + SECONDS_PER_MINUTE) // SECONDS_PER_MINUTE for p in valid_lines))
        table_width = max_pkg + max_type + PADDING_TIME_COL + PADDING_STATUS_COL + max_rt + PADDING_BORDERS
    else:
        table_width = WIDTH_FALLBACK
        
    # Initialize Console and Table with appropriate width and styling
    term_width = shutil.get_terminal_size().columns
    console = Console(width=max(term_width, table_width))
    table = Table(title="e2e Test Packages Runtime", show_header=True, header_style="bold magenta")
    for col, kwargs in [("Package Name", {"style": "cyan"}), ("Bucket Type", {"style": "blue"}), 
                        ("Time", {"justify": "right"}), ("Runtime (░=1m wait, ▓=1m run)", {}),
                        ("Status", {"justify": "center"})]: table.add_column(col, **kwargs)

    # Populate table rows
    for p in lines:
        if len(p) >= MIN_REQUIRED_FIELDS:
            code, start, end = int(p[IDX_EXIT_CODE]), int(p[IDX_START_TIME]), int(p[IDX_END_TIME])
            
            # Safely get the attempt count if it exists, default to 0 for backwards compatibility
            attempt = int(p[IDX_ATTEMPT]) if len(p) > IDX_ATTEMPT else 0
            
            wait, run = start // SECONDS_PER_MINUTE, (end - start + SECONDS_PER_MINUTE) // SECONDS_PER_MINUTE
            
            # Format status according to the new logic
            if code == 0:
                if attempt > 0:
                    status = f"[yellow]✅ FLAKY (Attempt {attempt})[/]"
                else:
                    status = "[green]✅ PASSED[/]"
            else:
                status = "[red]❌ FAILED[/]"
                
            table.add_row(p[IDX_PKG_NAME], p[IDX_BUCKET_TYPE], f"{run}m", f"[dim]{'░'*wait}[/][cyan]{'▓'*run}[/]", status)
    console.print(table)
    
except ImportError:
    print("Error: The 'rich' library is required to run this script. Please install it (e.g., 'pip install rich').", file=sys.stderr)
    sys.exit(1)
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
    sys.exit(1)
