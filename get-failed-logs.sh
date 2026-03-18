#!/bin/bash

VERSION="v999.999.987"
GCS_BASE="gs://gcsfuse-release-packages/$VERSION"

# Loop through all VM directories in the GCS bucket
for VM_URL in $(gcloud storage ls "$GCS_BASE/"); do
    # Extract the VM name (e.g., mky-release-test-debian-11)
    VM_NAME=$(basename "$VM_URL")

    # Loop through the e2e run directories inside that VM
    for RUN_URL in $(gcloud storage ls "$VM_URL" | grep 'gcsfuse-e2e-run-'); do
        # Extract the run ID
        RUN_NAME=$(basename "$RUN_URL")
        FAILED_LOGS_URL="${RUN_URL}failed_package_logs/"

        # Check if the failed_package_logs folder exists in this run
        if gcloud storage ls "$FAILED_LOGS_URL" >/dev/null 2>&1; then
            echo "🚨 Found failed logs in $VM_NAME. Downloading..."
            
            # Create the exact local structure (e.g., ./v999.../mky-.../gcsfuse-e2e-.../)
            LOCAL_DIR="./$VERSION/$VM_NAME/$RUN_NAME"
            mkdir -p "$LOCAL_DIR"
            
            # Copy *only* the failed logs folder
            gcloud storage cp -r "$FAILED_LOGS_URL" "$LOCAL_DIR/"
        else
            echo "✅ No failed logs for $VM_NAME. Skipping."
        fi
    done
done

echo "🎉 Done! All failed logs have been extracted."