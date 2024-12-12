
node_pool=anushkadhn-node-pool-300gb-boot-disk
instance_id=anushkadhn-filecacheoff-buffer-memory-boot-300gb
namespace=300gb

env project_id=tpu-prod-env-one-vm \
project_number=630405687483\
 zone=us-east5-c \
 cluster_name=anushkadhn-tpu-cluster \
 machine_type=ct6e-standard-4t \
 num_nodes=7 \
 use_custom_csi_driver=false \
 src_dir=/usr/local/google/home/anushkadhn/gcsfuse/.. \
  gcsfuse_branch=master \
   workload-config=perfmetrics/scripts/testing_on_gke/examples/workloads.json \
output_dir=. perfmetrics/scripts/testing_on_gke/examples/run-gke-tests.sh --debug   $namespace $node_pool $instance_id
