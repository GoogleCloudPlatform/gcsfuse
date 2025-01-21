# Num_nodes must be the number of ndoes in the nodepool which will be used.
cluster_name=$1
num_nodes=$2
nodepool=$3

bootDiskSize=100gb
buffer_location=memory
filecacheConfig=<Off/filecache/filecache-pd>
node_pool=$nodepool
instance_id=<anushkadhn>-filecache${filecacheConfig}-buffer-${buffer_location}-boot-${bootDiskSize}
namespace="${buffer_location}-${bootDiskSize}"

env project_id=tpu-prod-env-large-adhoc \
project_number=716203006749\
 zone=us-central2-b \
 cluster_name=${cluster_name} \
 machine_type=ct6e-standard-4t \
 num_nodes=${num_nodes} \
 use_custom_csi_driver=true \
 src_dir=/usr/local/google/home/anushkadhn/gcsfuse-custom-csi/.. \
 gcsfuse_branch=master \
 gcsfuse_src_dir=.\
 workload-config=./perfmetrics/scripts/testing_on_gke/examples/workloads.json \
output_dir=. perfmetrics/scripts/testing_on_gke/examples/run-gke-tests.sh --debug   $namespace $node_pool $instance_id $buffer_location



# Output csvs are stored under ./fio/output_{instance_id}.csv




