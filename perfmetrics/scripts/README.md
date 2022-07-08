# GCSFuse Metrics Extraction
This project will be used to run FIO load tests on a GCP VM and extract the GCSFuse metrics from these load tests into a Google Sheet. This can be used to monitor the performance of GCSFuse and make appropriate improvements in future.

## Installing required packages
### FIO
```bash
sudo apt-get update
sudo apt-get install fio -y
```
### GCSFuse
```bash
GCSFUSE_VERSION=0.41.1
curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v$GCSFUSE_VERSION/gcsfuse_"$GCSFUSE_VERSION"_amd64.deb
sudo dpkg --install gcsfuse_"$GCSFUSE_VERSION"_amd64.deb
```

## How to run:
1. Create a GCP VM with OS version as Ubuntu 20.04. Follow this [documentation](https://cloud.google.com/compute/docs/create-linux-vm-instance) and start your VM. Follow the next steps in your VM.
2. Install the required packages as mentioned in the above section
3. Clone the GCSFuse repo and cd into the perfmetrics/scripts directory:
```bash
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse/perfmetrics/scripts
```
4. Create a directory and mount your GCS bucket into it using GCSFuse:
```bash
mkdir -p your-directory-name
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --disable-http2"
BUCKET_NAME=your-bucket-name
MOUNT_POINT=your-directory-name
gcsfuse $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
```

5. Create a FIO job file `your-job-file.fio` and place it in the `job_files` directory
6. Run the FIO load test
```bash
fio job_files/your-job-file.fio --lat_percentiles 1 --output-format=json --output='output.json'
```
7. Install requirements by running `pip install -r requirements.txt --user`
8. Generate your service account credentials `creds.json` and upload the file on your GCS bucket `your-bucket-name`. If using an old credentials file, make sure that it is not expired. Run the following command to copy it into `gsheet` directory:
```bash
gsutil cp gs://your-bucket-name/creds.json ./gsheet
```
9. Create a Google Sheet with id `your-gsheet-id` by copying this Google Sheet: 
By default, cell `T4` contains the total number of entries in the worksheet.
10. Share the above copied Google Sheet with your service account(created in step 8)
11. Change the Google sheet id in this [line](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/perfmetrics/scripts/gsheet/gsheet.py#L5) to `your-gsheet-id`.
12. Finally, execute fetch_metrics.py to extract FIO and VM metrics and write to your Google Sheet. Pass the FIO output JSON file as argument to the fetch_metrics module.
`python3 fetch_metrics.py output.json`

## Adding new metrics

### FIO Metric
#### To add a new job parameter
1. Add another JobParam object to the REQ_JOB_PARAMS list. Make sure that json_name matches the key of the parameter in the FIO output JSON to avoid any errors.
2. Add a column in the Google Sheet in the same position as the position of the new parameter in the REQ_JOB_PARAMS list.
#### To add a new job metric
1. Add another JobMetric object to the REQ_JOB_METRICS list. Make sure that the values in LEVELS are correct and match the FIO output JSON keys in order to avoid any errors.
2. Add a column in the Google Sheet in the same position as the position of the new metric in the REQ_JOB_METRICS list

### VM Metric
#### If your metric data does not depend on the read/write type of FIO load test:
1. In the vm_metrics file, create an object of the [Metric class](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L44) and append the object to the [`METRICS_LIST`](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L63) list.
2. Add a column in the Google Sheet in the same position as the position of the new parameter in the [`METRICS_LIST`](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L63) list.

#### If your metric data depends on the read/write type of FIO load test:
1. In the vm_metrics file, create an object of the [Metric class](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L44) inside the [`_add_new_metric_using_test_type`](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L224) method and append the object to the [`updated_metrics_list`](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L245) list.
2. Add a column in the Google Sheet in the same position as the position of the new parameter in the [`updated_metrics_list`](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L245) list.

### Google Sheet
1. The number of entries are stored in `T4` cell of both worksheets. If on adding new metrics, the column number exceeds `T` in any worksheet, use another cell `cell-name` to store the number of entries by writing `=COUNTA(A2:A)` into it in both worksheets. Also, replace `T4` with `cell-name` in this [line](https://github.com/GoogleCloudPlatform/gcsfuse/blob/de3c0ab46f856bc1dbbfc6f093b1c218331499fe/perfmetrics/scripts/gsheet/gsheet.py#L9).
