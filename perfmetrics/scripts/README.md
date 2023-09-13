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
GCSFUSE_VERSION=0.41.4
curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v$GCSFUSE_VERSION/gcsfuse_"$GCSFUSE_VERSION"_amd64.deb
sudo dpkg --install gcsfuse_"$GCSFUSE_VERSION"_amd64.deb
```

## How to run
1. Create a GCP VM with OS version as Ubuntu 20.04. Follow this [documentation](https://cloud.google.com/compute/docs/create-linux-vm-instance) and start your VM. Follow the next steps in your VM.
2. Install the required packages as mentioned in the above section
3. Clone the GCSFuse repo and cd into the perfmetrics/scripts directory:
```bash
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse/perfmetrics/scripts
```
4. Create a directory and mount your GCS bucket into it using GCSFuse:
```bash
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --client-protocol http1"
BUCKET_NAME=your-bucket-name
mkdir -p your-directory-name
MOUNT_POINT=your-directory-name
gcsfuse $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
```

5. Create a FIO job file `your-job-file.fio` according to [this format](https://fio.readthedocs.io/en/latest/fio_doc.html#job-file-format) and place it in the `job_files` directory. Note that you can use only one job file. However, you can put any number of jobs within that job file. When there are multiple jobs in a FIO job file and a ```global startdelay``` is specified, the FIO metrics collection code expects that ```global startdelay``` is present between each of the jobs in one job file. To make sure the FIO load test has ```global startdelay``` amount of delay between each job, the following ```startdelay``` should be added to the job options in the FIO job file:

    1. For the first job the ```startdelay``` = ```global startdelay```

    2. For every subsequent job, ```startdelay``` = ```startdelay``` of previous job + ```runtime``` of previous job + ```ramp_time``` of previous job +       ```global startdelay```

6. Run the FIO load test
```bash
fio job_files/your-job-file.fio --lat_percentiles 1 --output-format=json --output='output.json'
```
7. Install requirements by running
```bash
pip install --require-hashes -r requirements.txt --user
```
8. Create a service account by following this [documentation](https://cloud.google.com/iam/docs/creating-managing-service-accounts). Generate your service account key, `creds.json` by following [this doc](https://cloud.google.com/iam/docs/creating-managing-service-account-keys#iam-service-account-keys-create-console) and upload the file on your GCS bucket `your-bucket-name`. If using an old credentials file, make sure that it is not expired. Run the following command to copy it into `gsheet` directory:
```bash
gsutil cp gs://your-bucket-name/creds.json ./gsheet
```
9. Create a Google Sheet with id `your-gsheet-id` by copying this [Google Sheet](https://docs.google.com/spreadsheets/d/1IJIjWuEs7cL6eYqPmlVaEGdclr6MSiaKJdnFXXC5tg8/).
10. Share the above copied Google Sheet with your service account(created in step 8)
11. Change the Google sheet id in this [line](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/perfmetrics/scripts/gsheet/gsheet.py#L5) to `your-gsheet-id`.
12. Finally, execute fetch_metrics.py to extract FIO and VM metrics and write to your Google Sheet by running
```bash
python3 fetch_metrics.py output.json
```
The FIO output JSON file is passed as an argument to the fetch_metrics module.

### Note

The previous data in the google sheet will be deleted every time you enter new data. Therefore, at any point of time, the google sheet will store only the last tests’ data. If you want, you can change this in the [```gsheet/gsheet.py```](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/perfmetrics/scripts/gsheet/gsheet.py) file.

## Adding new metrics

### FIO Metric
#### To add a new job parameter
1. Add another object of [JobParam class](https://github.com/GoogleCloudPlatform/gcsfuse/blob/be488de374db77748813a5bc7d710cf9eed425d7/perfmetrics/scripts/fio/fio_metrics.py#L23) to the REQ_JOB_PARAMS list [here](https://github.com/GoogleCloudPlatform/gcsfuse/blob/a454b452f5fd290f9ef3cc0da85b9d27d6beee4a/perfmetrics/scripts/fio/fio_metrics.py#L76). Make sure that json_name matches the key of the parameter in the FIO output JSON to avoid any errors.
2. Add a column in the Google Sheet in the same position as the position of the new parameter in the REQ_JOB_PARAMS list.
#### To add a new job metric
1. Add another object of [JobMetric class](https://github.com/GoogleCloudPlatform/gcsfuse/blob/be488de374db77748813a5bc7d710cf9eed425d7/perfmetrics/scripts/fio/fio_metrics.py#L48) to the REQ_JOB_METRICS list [here](https://github.com/GoogleCloudPlatform/gcsfuse/blob/a454b452f5fd290f9ef3cc0da85b9d27d6beee4a/perfmetrics/scripts/fio/fio_metrics.py#L97). Make sure that the values in LEVELS are correct and match the FIO output JSON keys in order to avoid any errors.
2. Add a column in the Google Sheet in the same position as the position of the new metric in the REQ_JOB_METRICS list

### VM Metric
#### If your metric data does not depend on the read/write type of FIO load test
1. In the vm_metrics file, create an object of the [Metric class](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L44) and append the object to the [`METRICS_LIST`](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L63) list.
2. Add a column in the Google Sheet in the same position as the position of the new parameter in the [`METRICS_LIST`](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L63) list.

#### If your metric data depends on the read/write type of FIO load test
1. In the vm_metrics file, create an object of the [Metric class](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L44) inside the [`_add_new_metric_using_test_type`](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L224) method and append the object to the [`updated_metrics_list`](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L245) list.
2. Add a column in the Google Sheet in the same position as the position of the new parameter in the [`updated_metrics_list`](https://github.com/GoogleCloudPlatform/gcsfuse/blob/fbe86d40bdefefc1595654fa468a81e4dfd815d5/perfmetrics/scripts/vm_metrics/vm_metrics.py#L245) list.
