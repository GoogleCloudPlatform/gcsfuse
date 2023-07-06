# Listing Operation Benchmarking Script

This script is used to benchmark the performance (latency) of listing operation in GCSFuse mounted bucket. Further a side by side comparision of GCSFuse performance is made with the persistent disk.\
The main file of this project is [listing_benchmark.py](listing_benchmark.py) python script. This script by itself creates the necessary directory structure, containing files and folders, needed to test the listing operation. Then it systematically test the operation on the directory structure and parse the results. Also if required it can upload the results of the test to a Google Sheet.\
It takes input a JSON config file which contains the info regarding directory structure and also through which multiple tests of different configurations can be performed in a single run.

## Installing required packages

### GCSFuse
```
GCSFUSE_VERSION=0.41.4
curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v$GCSFUSE_VERSION/gcsfuse_"$GCSFUSE_VERSION"_amd64.deb
sudo dpkg --install gcsfuse_"$GCSFUSE_VERSION"_amd64.deb
```

### gsutil interface
Please refer to the official documentation of gsutil [here](https://cloud.google.com/storage/docs/gsutil_install) to install it.

### Required python modules
```
pip install --require-hashes -r requirements.txt --user
```

## Flags to use with the python script
1. Flag -h: Typical help interface of the script.
2. Flag --keep_files: Do not delete the generated directory structure from the persistent disk after running the tests.
3. Flag --upload_gs: Uploads the results of the test to the Google Sheet.
4. Flag --upload_bq: Uploads the results of the test to the BiqQuery.
5. Flag --num_samples: Runs each test for NUM_SAMPLES times.
6. Flag --message: Takes input a message string, which describes/titles the test.
7. Flag --config_id: Configuration id of the experiment in BigQuery tables.
8. Flag --start_time_build: Time at which KOKORO triggered the build scripts
8. Flag --gcsfuse_flags (required): GCSFUSE flags with which the list tests bucket will be mounted. 
9. Flag --command (required): Takes a input a string, which is the command to run the tests on.  
10. config_file (required): Path to the JSON config file which contains the details of the tests.

## How to run
1. Create a GCP VM with OS version as Ubuntu 20.04. Follow this [documentation](https://cloud.google.com/compute/docs/create-linux-vm-instance) and start your VM.
2. Install the required packages as mentioned in the above section.
3. Create a service account by following this [documentation](https://cloud.google.com/iam/docs/creating-managing-service-accounts). Generate your service account key, `creds.json` by following [this doc](https://cloud.google.com/iam/docs/creating-managing-service-account-keys#iam-service-account-keys-create-console) and upload the file on your GCS bucket `your-bucket-name`. If using an old credentials file, make sure that it is not expired. Run the following command to copy it into `gsheet` directory:
```bash
gsutil cp gs://your-bucket-name/creds.json ../gsheet
```
4. Create a Google Sheet with id `your-gsheet-id` by copying this [Google Sheet](https://docs.google.com/spreadsheets/d/1IJIjWuEs7cL6eYqPmlVaEGdclr6MSiaKJdnFXXC5tg8/).
5. Share the above copied Google Sheet with your service account(created in step 2)
6. Change the Google sheet id in this [line](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/perfmetrics/scripts/gsheet/gsheet.py#L5) to `your-gsheet-id`.
7. Configure the [JSON config file](config.json) as per needs.
8. Run the custom python script. A sample command is shown below:
```
python3 listing_benchmark.py config.json --command "ls -R" --upload
```

**Note**: Steps 3, 4, 5, and 6 are needed only if you want to upload results to the Google Sheet.
