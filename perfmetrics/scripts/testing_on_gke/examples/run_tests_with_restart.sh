#!/bin/bash
#set -e
export project_id=gcs-fuse-test
export project_number=927584127901
export zone="us-west1-c"
export cluster_name="princer-n2-us-west1-c"
export machine_type="n2-standard-96"
export num_nodes=8
export num_ssd="16"
export use_custom_csi_driver="true"
export gcsfuse_src_dir=/home/princer_google_com/pd_gke_testing/gcsfuse
export csi_src_dir=/home/princer_google_com/pd_gke_testing/gcs-fuse-csi-driver
export output_dir=./


gcloud container clusters delete --quiet --zone $zone $cluster_name
export workload_config=$gcsfuse_src_dir/perfmetrics/scripts/testing_on_gke/examples/parallel_downloads/all_combo_normal.yaml
export instance_id=
export only_parse=""
./run-gke-tests.sh --debug

sleep 90s
gcloud container clusters delete --quiet --zone $zone $cluster_name
export workload_config=$gcsfuse_src_dir/perfmetrics/scripts/testing_on_gke/examples/parallel_downloads/all_combo_pd.yaml
export instance_id=<your unique id 2>
export only_parse=""
./run-gke-tests.sh --debug

sleep 90s
gcloud container clusters delete --quiet --zone $zone $cluster_name
export workload_config=$gcsfuse_src_dir/perfmetrics/scripts/testing_on_gke/examples/parallel_downloads/all_combo_file_cache.yaml
export instance_id=<your unique id 3>
export only_parse=""
./run-gke-tests.sh --debug

sleep 90s
gcloud container clusters delete --quiet --zone $zone $cluster_name
export workload_config=$gcsfuse_src_dir/perfmetrics/scripts/testing_on_gke/examples/parallel_downloads/10g_normal.yaml
export instance_id=<your unique id 4>
export only_parse=""
./run-gke-tests.sh --debug

sleep 90s
gcloud container clusters delete --quiet --zone $zone $cluster_name
export workload_config=$gcsfuse_src_dir/perfmetrics/scripts/testing_on_gke/examples/parallel_downloads/10g_file_cache.yaml
export instance_id=<your unique id 5>
export only_parse=""
./run-gke-tests.sh --debug

sleep 90s
gcloud container clusters delete --quiet --zone $zone $cluster_name
export workload_config=$gcsfuse_src_dir/perfmetrics/scripts/testing_on_gke/examples/parallel_downloads/10g_pd.yaml
export instance_id=<your unique id 6>
export only_parse=""
./run-gke-tests.sh --debug
