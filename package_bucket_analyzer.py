#!/usr/bin/env python3

import os
import sys
import re
from pathlib import Path

def analyze_test_level_logs(parent_dir):
    # Data structure: results[package_name][bucket_type][test_name] = {'passed': 0, 'failed': 0, 'skipped': 0}
    results = {}
    target_folders = ['failed_package_logs', 'success_package_logs']
    files_processed = 0

    # Regex to match Go's test output lines and extract the test name
    # Matches: "--- PASS: TestName (0.00s)" or "   --- FAIL: SubTest/Name (0.12s)"
    test_pattern = re.compile(r'---\s+(PASS|FAIL|SKIP):\s+([^\s]+)')

    print(f"Scanning '{parent_dir}' for individual test results...")

    for filepath in Path(parent_dir).rglob('*.txt'):
        parts = filepath.parts
        log_type = None
        for tf in target_folders:
            if tf in parts:
                log_type = tf
                break
                
        if not log_type:
            continue
            
        bucket_type = filepath.parent.name
        package_name = filepath.stem
        
        # Initialize nested dictionaries
        if package_name not in results:
            results[package_name] = {}
        if bucket_type not in results[package_name]:
            results[package_name][bucket_type] = {}

        files_processed += 1
        has_parsed_tests = False
        has_failures = False

        try:
            with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
                for line in f:
                    match = test_pattern.search(line)
                    if match:
                        has_parsed_tests = True
                        status = match.group(1).upper()
                        test_name = match.group(2)
                        
                        if test_name not in results[package_name][bucket_type]:
                            results[package_name][bucket_type][test_name] = {'passed': 0, 'failed': 0, 'skipped': 0}
                            
                        if status == 'PASS':
                            results[package_name][bucket_type][test_name]['passed'] += 1
                        elif status == 'FAIL':
                            results[package_name][bucket_type][test_name]['failed'] += 1
                            has_failures = True
                        elif status == 'SKIP':
                            results[package_name][bucket_type][test_name]['skipped'] += 1
                            
            # Edge Case: If the package log was dumped in the 'failed' directory but no specific 
            # '--- FAIL:' lines were printed, the test binary panicked, failed to compile, or timed out.
            if log_type == 'failed_package_logs' and not has_failures:
                dummy_name = "<PACKAGE_CRASH_OR_TIMEOUT>"
                if dummy_name not in results[package_name][bucket_type]:
                    results[package_name][bucket_type][dummy_name] = {'passed': 0, 'failed': 0, 'skipped': 0}
                results[package_name][bucket_type][dummy_name]['failed'] += 1

        except Exception as e:
            print(f"Error reading {filepath}: {e}")

    return results, files_processed

def generate_package_bucket_report(results, files_processed):
    print("\n" + "=" * 115)
    print(" GCSFUSE E2E PACKAGE x BUCKET TYPE ANALYSIS ".center(115, "="))
    print("=" * 115)
    print(f"Total Log Files Processed: {files_processed}")
    print("-" * 115)

    if not results:
        print("\nNo detailed test logs found. Ensure you are pointing to the correct root directory.")
        return

    # Iterate through packages alphabetically
    for package_name in sorted(results.keys()):
        # Iterate through bucket types (flat, hns) for the current package
        for bucket_type in sorted(results[package_name].keys()):
            tests = results[package_name][bucket_type]
            
            # Print the Header for this Package x Bucket combination
            print(f"\n" + "=" * 115)
            print(f" PACKAGE: {package_name.upper()} | BUCKET TYPE: {bucket_type.upper()} ".center(115, "="))
            print("=" * 115)
            
            stats = []
            for test_name, counts in tests.items():
                passed = counts['passed']
                failed = counts['failed']
                skipped = counts['skipped']
                
                # Calculate execution rate (ignoring skips for accurate pass/fail metrics)
                total_exec = passed + failed
                fail_rate = (failed / total_exec * 100) if total_exec > 0 else 0
                
                stats.append({
                    'test': test_name,
                    'total_exec': total_exec,
                    'passed': passed,
                    'failed': failed,
                    'skipped': skipped,
                    'fail_rate': fail_rate
                })
            
            # Sort tests: Highest failure rate first, then by highest total failures, then alphabetically
            stats.sort(key=lambda x: (x['fail_rate'], x['failed']), reverse=True)
            
            # Define table structure for the current block
            header = f"| {'Specific Test Name':<65} | {'Total':<7} | {'Passed':<6} | {'Failed':<8} | {'Skipped':<7} | {'Fail %':<8} |"
            print("-" * len(header))
            print(header)
            print("-" * len(header))
            
            # Print rows
            for s in stats:
                fail_pct_str = f"{s['fail_rate']:.1f}%"
                
                if s['failed'] > 0:
                    fail_str = f"{s['failed']} ❌"
                    fail_pct_str = f"{fail_pct_str} ⚠️"
                else:
                    fail_str = str(s['failed'])
                    
                # Truncate extremely long names so they don't break the ASCII table alignment
                test_str = (s['test'][:62] + '...') if len(s['test']) > 65 else s['test']
                
                row = f"| {test_str:<65} | {s['total_exec']:<7} | {s['passed']:<6} | {fail_str:<8} | {s['skipped']:<7} | {fail_pct_str:<8} |"
                print(row)
                
            print("-" * len(header))

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python3 package_bucket_analyzer.py <path_to_parent_directory>")
        sys.exit(1)
        
    parent_directory = sys.argv[1]
    
    if not os.path.isdir(parent_directory):
        print(f"Error: Directory '{parent_directory}' does not exist.")
        sys.exit(1)
        
    results, processed_count = analyze_test_level_logs(parent_directory)
    generate_package_bucket_report(results, processed_count)