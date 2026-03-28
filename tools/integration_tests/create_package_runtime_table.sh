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
    import subprocess
    
    venv_dir = "/tmp/.gcsfuse_rich_venv"
    python_exe = os.path.join(venv_dir, "bin", "python3")
    pip_exe = os.path.join(venv_dir, "bin", "pip")
    if sys.executable != python_exe:
        print("Setting up rich CLI for beautiful reporting...", file=sys.stderr)
        if not os.path.exists(python_exe):
            subprocess.run([sys.executable, "-m", "venv", venv_dir], check=True)
            
        subprocess.run([pip_exe, "install", "-i", "https://pypi.org/simple", "--quiet", "rich"], check=True)
        # Re-execute the script in the newly created venv
        os.execv(python_exe, [python_exe] + sys.argv)
    else:
        print("Error: The 'rich' library failed to install or load. Cannot generate the table.", file=sys.stderr)
        sys.exit(1)
