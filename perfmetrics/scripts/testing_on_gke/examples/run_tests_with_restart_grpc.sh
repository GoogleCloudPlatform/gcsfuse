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
export only_parse=""

gcloud container clusters delete --quiet --zone $zone $cluster_name
export workload_config=$gcsfuse_src_dir/perfmetrics/scripts/testing_on_gke/examples/parallel_downloads_grpc/all_combo_normal.yaml
export instance_id=princer_grpc_combo_normal
./run-gke-tests.sh --debug

sleep 90s
gcloud container clusters delete --quiet --zone $zone $cluster_name
export workload_config=$gcsfuse_src_dir/perfmetrics/scripts/testing_on_gke/examples/parallel_downloads_grpc/all_combo_file_cache.yaml
export instance_id=princer_grpc_combo_file_cache
./run-gke-tests.sh --debug

sleep 90s
gcloud container clusters delete --quiet --zone $zone $cluster_name
export workload_config=$gcsfuse_src_dir/perfmetrics/scripts/testing_on_gke/examples/parallel_downloads_grpc/all_combo_pd.yaml
export instance_id=princer_grpc_combo_pd
./run-gke-tests.sh --debug

sleep 90s
gcloud container clusters delete --quiet --zone $zone $cluster_name
export workload_config=$gcsfuse_src_dir/perfmetrics/scripts/testing_on_gke/examples/parallel_downloads_grpc/10g_normal.yaml
export instance_id=princer_grpc_10g_normal
./run-gke-tests.sh --debug

sleep 90s
gcloud container clusters delete --quiet --zone $zone $cluster_name
export workload_config=$gcsfuse_src_dir/perfmetrics/scripts/testing_on_gke/examples/parallel_downloads_grpc/10g_file_cache.yaml
export instance_id=princer_grpc_10g_file_cache
./run-gke-tests.sh --debug

sleep 90s
gcloud container clusters delete --quiet --zone $zone $cluster_name
export workload_config=$gcsfuse_src_dir/perfmetrics/scripts/testing_on_gke/examples/parallel_downloads_grpc/10g_pd.yaml
export instance_id=princer_grpc_10g_pd
./run-gke-tests.sh --debug
