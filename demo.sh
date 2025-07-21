# Step: 1: Mounts GCS bucket using GCSFuse.
echo "********************** Mounting GCS bucket using GCSFuse **********************"
(umount ~/gcs || true)  && go install . && gcsfuse princer-read-cache-load-test-west ~/gcs

# Step: 2: Runs the fio job containing a random read workload with 48 threads, where each thread reads 5 GiB file randomly.
echo "********************** Running a random read workload using FIO on GCSFuse Mounted Directory **********************"
cd ~/gcs && fio /home/princer_google_com/dev/gcsfuse/perfmetrics/scripts/job_files/read_cache_load_test.fio && cd - && sleep 15 && umount ~/gcs

# Step: 3: GCSFuse generates a workload insight for the previous fio workload.
echo "********************** Generated workload insight **********************"
cat ~/workload_insight.yaml

# Step: 4: Pass the generated workload insight to the Gemini model to generate GCSFuse config.
export GEMINI_API_KEY="<add_token_api>"
echo "********************** Feeding the workload insight to gemini model to generate GCSFuse config **********************"
cd ~/dev/go-core/ai && go run main.go && cd -

# Step: 5: Print the generated config on console.
echo "********************** Generated GCSFuse config **********************"
cat ~/go-core/ai/generated_gcsfuse_config.yaml