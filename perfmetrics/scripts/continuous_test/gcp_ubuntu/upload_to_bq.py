import os
import sys
import json
import logging
import datetime
import argparse
from google.cloud import bigquery, exceptions

logging.basicConfig(level=logging.INFO, format='%(levelname)s: %(message)s')

def upload_results_to_bq(fio_json_path, is_kokoro):
    project_id = "gcs-fuse-test-ml"
    
    # Dataset logic per your request
    if is_kokoro:
        dataset_id = "periodic_benchmarks_trial"
        table_prefix = "kokoro_run"
    else:
        dataset_id = "adhoc_benchmarks_trial"
        table_prefix = "local_run"

    # Generate unique table ID with timestamp
    table_id = f"{table_prefix}_{datetime.datetime.now().strftime('%Y%m%d_%H%M%S')}"
    client = bigquery.Client(project=project_id)
    full_table_id = f"{project_id}.{dataset_id}.{table_id}"
    
    try:
        # 1. Ensure Dataset exists
        dataset_ref = client.dataset(dataset_id)
        try:
            client.get_dataset(dataset_ref)
        except exceptions.NotFound:
            logging.info(f"Creating dataset {dataset_id}")
            client.create_dataset(bigquery.Dataset(dataset_ref))

        # 2. Define Schema
        schema = [
            bigquery.SchemaField("run_timestamp", "TIMESTAMP", mode="REQUIRED"),
            bigquery.SchemaField("iteration", "INTEGER", mode="REQUIRED"),
            bigquery.SchemaField("gcsfuse_flags", "STRING"),
            bigquery.SchemaField("fio_json_output", "JSON"),
        ]

        # 3. Create Table
        table = bigquery.Table(full_table_id, schema=schema)
        client.create_table(table)

        # 4. Load FIO JSON
        with open(fio_json_path, 'r') as f:
            fio_data = json.load(f)

        row = {
            "run_timestamp": datetime.datetime.utcnow().isoformat(),
            "iteration": 1,
            "gcsfuse_flags": "master",
            "fio_json_output": json.dumps(fio_data)
        }

        errors = client.insert_rows_json(full_table_id, [row])
        if errors:
            logging.error(f"Insert errors: {errors}")
            sys.exit(1)
        
        # Standardized output for build.sh to grab
        print(f"RESULT_TABLE_ID={table_id}")

    except Exception as e:
        logging.error(f"Failed to upload to BQ: {e}")
        sys.exit(1)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--result_file", required=True)
    parser.add_argument("--kokoro", action="store_true")
    args = parser.parse_args()
    upload_results_to_bq(args.result_file, args.kokoro)