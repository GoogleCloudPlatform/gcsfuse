#!/usr/bin/env python3

import os
import sys
from pathlib import Path

def analyze_logs(parent_dir):
    # Data structure: results[bucket_type][package_name] = {'passed': 0, 'failed': 0}
    results = {}
    files_processed = 0
    missing_table_files = []

    # Find all logs.txt files in the given directory and subdirectories
    for filepath in Path(parent_dir).rglob('logs.txt'):
        files_processed += 1
        found_table = False
        
        try:
            with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
                in_table = False
                for line in f:
                    line = line.strip()
                    
                    # Detect the start of the summary table
                    if "Package Name" in line and "Bucket Type" in line and "Status" in line:
                        in_table = True
                        found_table = True
                        continue
                        
                    if in_table:
                        # Stop parsing if we hit a blank line or the table ends
                        if not line or (not line.startswith('|') and not line.startswith('+')):
                            in_table = False
                            continue
                            
                        # Ignore table separators
                        if line.startswith('+'):
                            continue
                            
                        # Parse the table row
                        # Example: | benchmarking | flat | ✅PASSED | 1m | ...
                        parts = [p.strip() for p in line.split('|')]
                        if len(parts) >= 4 and parts[1]:
                            pkg = parts[1]
                            btype = parts[2]
                            status_raw = parts[3]
                            
                            if btype not in results:
                                results[btype] = {}
                            if pkg not in results[btype]:
                                results[btype][pkg] = {'passed': 0, 'failed': 0}
                                
                            if 'PASSED' in status_raw:
                                results[btype][pkg]['passed'] += 1
                            elif 'FAILED' in status_raw:
                                results[btype][pkg]['failed'] += 1
        except Exception as e:
            print(f"Error reading {filepath}: {e}")
            
        if not found_table:
            missing_table_files.append(str(filepath))

    return results, files_processed, missing_table_files

def generate_report(results, files_processed, missing_table_files):
    print("=" * 80)
    print(" GCSFUSE E2E TEST ANALYSIS REPORT ".center(80, "="))
    print("=" * 80)
    print(f"Total Log Files Processed: {files_processed}")
    if missing_table_files:
        print(f"Warning: {len(missing_table_files)} log files did not contain a final summary table (possible crashes).")
    print("-" * 80)

    if not results:
        print("\nNo test results found. Please check the directory path.")
        return

    # Process each bucket type separately
    for btype, packages in sorted(results.items()):
        print(f"\n>> BUCKET TYPE: [{btype.upper()}] <<\n")
        
        # Calculate stats and sort
        stats = []
        for pkg, counts in packages.items():
            passed = counts['passed']
            failed = counts['failed']
            total = passed + failed
            fail_rate = (failed / total * 100) if total > 0 else 0
            stats.append({
                'pkg': pkg,
                'total': total,
                'passed': passed,
                'failed': failed,
                'fail_rate': fail_rate
            })
            
        # Sort by Failure Rate (Highest to Lowest), then by Total Failed
        stats.sort(key=lambda x: (x['fail_rate'], x['failed']), reverse=True)
        
        # Print Table Header
        header = f"| {'Package Name':<30} | {'Total Runs':<10} | {'Passed':<8} | {'Failed':<8} | {'Failure %':<10} |"
        print("-" * len(header))
        print(header)
        print("-" * len(header))
        
        # Print Table Rows
        for s in stats:
            fail_pct_str = f"{s['fail_rate']:.1f}%"
            # Highlight failures for better readability
            if s['failed'] > 0:
                fail_str = f"{s['failed']} ❌"
                fail_pct_str = f"{fail_pct_str} ⚠️"
            else:
                fail_str = str(s['failed'])
                
            row = f"| {s['pkg']:<30} | {s['total']:<10} | {s['passed']:<8} | {fail_str:<8} | {fail_pct_str:<10} |"
            print(row)
            
        print("-" * len(header))

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python3 analyze_logs.py <path_to_parent_directory>")
        sys.exit(1)
        
    parent_directory = sys.argv[1]
    
    if not os.path.isdir(parent_directory):
        print(f"Error: Directory '{parent_directory}' does not exist.")
        sys.exit(1)
        
    print(f"Scanning '{parent_directory}' for log files...")
    results, processed_count, missing = analyze_logs(parent_directory)
    generate_report(results, processed_count, missing)