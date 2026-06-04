#!/usr/bin/env python3
# Copyright 2026 Google LLC
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

import argparse
import subprocess
import os
import tempfile
import shutil
from datetime import datetime

def run_cmd(cmd, check=True):
    res = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    if check and res.returncode != 0:
        raise Exception(f"Command failed: {cmd}\nStdout: {res.stdout}\nStderr: {res.stderr}")
    return res

def get_commit_metadata(commit_sha):
    try:
        date_res = run_cmd(f"git show -s --format=%cI {commit_sha}")
        author_res = run_cmd(f"git show -s --format=%an {commit_sha}")
        subject_res = run_cmd(f"git show -s --format=%s {commit_sha}")
        return {
            "date": date_res.stdout.strip(),
            "author": author_res.stdout.strip(),
            "subject": subject_res.stdout.strip()
        }
    except Exception:
        return {
            "date": datetime.utcnow().isoformat() + "Z",
            "author": "unknown",
            "subject": "unknown"
        }

def get_coverage_for_dirs(dirs):
    valid_dirs = []
    for d in dirs:
        if os.path.isdir(d) and len([f for f in os.listdir(d) if f.startswith("covcounters.")]) > 0:
            valid_dirs.append(d)
    
    if not valid_dirs:
        return 0.0

    temp_dir = tempfile.mkdtemp()
    try:
        merged_dir = os.path.join(temp_dir, "merged")
        os.makedirs(merged_dir, exist_ok=True)
        joined_dirs = ",".join(valid_dirs)
        
        # Merge
        run_cmd(f"go tool covdata merge -i={joined_dirs} -o={merged_dir}")
        
        # Text format
        text_profile = os.path.join(temp_dir, "coverage.out")
        run_cmd(f"go tool covdata textfmt -i={merged_dir} -o={text_profile}")
        
        # Parse coverage
        total_stmts = 0
        covered_stmts = 0
        with open(text_profile, "r") as f:
            f.readline()  # skip mode line
            for line in f:
                line = line.strip()
                if not line:
                    continue
                parts = line.split()
                if len(parts) != 3:
                    continue
                _, num_stmts_str, count_str = parts
                try:
                    num_stmts = int(num_stmts_str)
                    count = int(count_str)
                except ValueError:
                    continue
                total_stmts += num_stmts
                if count > 0:
                    covered_stmts += num_stmts

        pct = (covered_stmts / total_stmts * 100.0) if total_stmts > 0 else 0.0
        return round(pct, 2)
    finally:
        shutil.rmtree(temp_dir)

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--unit-dir", required=True, help="Directory containing raw unit test counters")
    parser.add_argument("--zonal-dir", required=True, help="Directory containing raw zonal E2E counters")
    parser.add_argument("--regional-dir", required=True, help="Directory containing raw regional E2E counters")
    
    parser.add_argument("--project", default="", help="BigQuery GCP Project ID")
    parser.add_argument("--dataset", default="gcsfuse_coverage", help="BigQuery Dataset Name")
    parser.add_argument("--table", default="trends", help="BigQuery Table Name")
    args = parser.parse_args()

    commit_sha = run_cmd("git rev-parse HEAD").stdout.strip()
    meta = get_commit_metadata(commit_sha)

    print("Calculating Unit coverage...")
    cov_unit = get_coverage_for_dirs([args.unit_dir])
    
    print("Calculating E2E/Integration coverage...")
    cov_e2e = get_coverage_for_dirs([args.zonal_dir, args.regional_dir])
    
    print("Calculating Combined (Unit + E2E) coverage...")
    cov_e2e_unit = get_coverage_for_dirs([args.unit_dir, args.zonal_dir, args.regional_dir])

    print(f"\nResults for commit {commit_sha[:8]}:")
    print(f"  Unit:          {cov_unit}%")
    print(f"  E2E:           {cov_e2e}%")
    print(f"  Unit + E2E:    {cov_e2e_unit}%")

    bq_timestamp = meta["date"]
    author = meta["author"].replace("'", "\\'")
    subject = meta["subject"].replace("'", "\\'")

    query = f"""
    INSERT INTO `{args.dataset}.{args.table}` 
    (timestamp, commit_sha, author, subject, coverage_e2e_unit, coverage_unit, coverage_e2e) 
    VALUES 
    (TIMESTAMP('{bq_timestamp}'), '{commit_sha}', '{author}', '{subject}', {cov_e2e_unit}, {cov_unit}, {cov_e2e})
    """

    print(f"\nInjecting results into BigQuery table '{args.dataset}.{args.table}'...")
    bq_project_opt = f"--project_id={args.project}" if args.project else ""
    
    bq_cmd = f"bq query {bq_project_opt} --use_legacy_sql=false \"{query}\""
    
    try:
        run_cmd(bq_cmd)
        print("Success! BigQuery timeline updated successfully.")
    except Exception as e:
        print(f"\nFailed to inject data to BigQuery: {e}")
        print("Please verify that the dataset and table exist in BigQuery and the schema matches:")
        print("  Schema: timestamp:TIMESTAMP, commit_sha:STRING, author:STRING, subject:STRING, coverage_e2e_unit:FLOAT, coverage_unit:FLOAT, coverage_e2e:FLOAT")

if __name__ == "__main__":
    main()
