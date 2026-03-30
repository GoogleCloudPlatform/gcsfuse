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
    <package_name> <bucket_type> <exit_code> <start_time_seconds> <end_time_seconds>

Example File Content:
    pkg_name bucket_type exit_code start_time_seconds end_time_seconds
    pkg1 bucket-standard 0 0 120
    pkg2 bucket-premium 0 60 180
    pkg3 bucket-standard 1 120 240
"""
import sys, os

if len(sys.argv) != 2:
    print(f"Usage: {sys.argv[0]} <FILE_PATH>")
    sys.exit(1)

path = sys.argv[1]
if not os.path.isfile(path):
    print(f"Error: File '{path}' not found.")
    sys.exit(1)

with open(path) as f:
    lines = sorted([l.split() for l in f if l.strip()])

try:
    from rich.console import Console
    from rich.table import Table
    import shutil
    valid_lines = [p for p in lines if len(p) >= 5]
    if valid_lines:
        max_pkg = max(12, max(len(p[0]) for p in valid_lines))
        max_type = max(11, max(len(p[1]) for p in valid_lines))
        max_rt = max(31, max(int(p[3])//60 + (int(p[4])-int(p[3])+60)//60 for p in valid_lines))
        table_width = max_pkg + max_type + 8 + 10 + max_rt + 20
    else:
        table_width = 80
        
    term_width = shutil.get_terminal_size().columns
    console = Console(width=max(term_width, table_width))
    table = Table(title="e2e Test Packages Runtime", show_header=True, header_style="bold magenta")
    for col, kwargs in [("Package Name", {"style": "cyan"}), ("Bucket Type", {"style": "blue"}), 
                        ("Time", {"justify": "right"}), ("Runtime (░=1m wait, ▓=1m run)", {}),
                        ("Status", {"justify": "center"})]: table.add_column(col, **kwargs)

    for p in lines:
        if len(p) >= 5:
            code, start, end = int(p[2]), int(p[3]), int(p[4])
            wait, run = start // 60, (end - start + 60) // 60
            status = "[green]✅ PASSED[/]" if code == 0 else "[red]❌ FAILED[/]"
            table.add_row(p[0], p[1], f"{run}m", f"[dim]{'░'*wait}[/][cyan]{'▓'*run}[/]", status)
    console.print(table)
    
except ImportError:
    print("Error: The 'rich' library is required to run this script. Please install it (e.g., 'pip install rich').", file=sys.stderr)
    sys.exit(1)
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
    sys.exit(1)
