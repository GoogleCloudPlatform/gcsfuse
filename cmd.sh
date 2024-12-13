bootDiskSize=300gb
buffer_location=boot
filecacheConfig=Off
node_pool=anushkadhn-node-pool-${bootDiskSize}-boot-disk
instance_id=anushkadhn-filecacheoff-buffer-boot-boot-${bootDiskSize}
namespace=${bootDiskSize}

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


path=$(pwd)
gsutil cp "${path}/fio/output_${instance_id}.csv" gs://anushkadhn-test/$bootDiskSize/buffer-on-${buffer_location}/fio/${filecacheConfig}.csv
gsutil cp "${path}/dlio/output_${instance_id}.csv" gs://anushkadhn-test/$bootDiskSize/buffer-on-${buffer_location}/dlio/${filecacheConfig}.csv