#!/bin/bash

set -e
set -x

function install_deps() {
    sudo apt-get update
    sudo apt-get install -y fio
}

# Create bucket
function create_bucket() {
    # Zonal bucket.
    gcloud storage buckets create gs://fastbyte-team-princer-zb-write-test-uw4a --location=us-west4 --default-storage-class=RAPID --enable-hierarchical-namespace --placement us-west4-a  --uniform-bucket-level-access --project=gcs-tess
    
    # GRPC bucket.
    gcloud storage buckets create gs://princer-grpc-write-test-uw4a --location=us-west4 --default-storage-class=STANDARD --enable-hierarchical-namespace --uniform-bucket-level-access --project=gcs-tess
}

function install_go_and_add_in_path() {
    version=1.23.4
    wget -O go_tar.tar.gz https://go.dev/dl/go${version}.linux-amd64.tar.gz -q
    sudo rm -rf /usr/local/go
    tar -xzf go_tar.tar.gz && sudo mv go /usr/local
    export PATH=$PATH:/usr/local/go/bin && go version && rm go_tar.tar.gz

    # Add go in the path permanently, so that $HOME/go/bin is visible.
    export PATH=$PATH:$HOME/go/bin/
    echo 'export PATH=$PATH:$HOME/go/bin/:/usr/local/go/bin' >> ~/.bashrc
}

function install_gcsfuse() {
    git clone https://github.com/GoogleCloudPlatform/gcsfuse.git && cd ./gcsfuse && go install
}

function init() {

    
    export TEST_BUCKET_ZONAL=fastbyte-team-princer-zb-write-test-uw4a
    export TEST_BUCKET_GRPC=princer-grpc-write-test-uw4a

    mkdir -p ~/logs

    mkdir -p ~/bucket-grpc
    gcsfuse --config-file ~/dev/gcsfuse-tools/write-test/config.yaml $TEST_BUCKET_GRPC "~/bucket-grpc"
    cd ~/bucket-grpc && mkdir ./256K && mkdir ./1M && mkdir ./120M && mkdir ./500M && mkdir ./1G && mkdir ./2G

    mkdir -p ~/bucket-zonal
    gcsfuse --config-file ~/dev/gcsfuse-tools/write-test/config.yaml $TEST_BUCKET_ZONAL "~/bucket-zonal"
    cd ~/bucket-zonal && mkdir ./256K && mkdir ./1M && mkdir ./120M && mkdir ./500M && mkdir ./1G && mkdir ./2G
}


function run() {
    export TEST_BUCKET_ZONAL=fastbyte-team-princer-zb-write-test-uw4a
    export TEST_BUCKET_GRPC=princer-grpc-write-test-uw4a

    # gcsfuse --config-file ~/dev/gcsfuse-tools/write-test/config.yaml $TEST_BUCKET_ZONAL "~/bucket-zonal"
    # cd ~/bucket-zonal
    # ~/dev/gcsfuse-tools/write-test/write_master.sh
    # cd -
    # umount ~/bucket-zonal && sleep 50
    
    gcsfuse --config-file ~/dev/gcsfuse-tools/write-test/config.yaml $TEST_BUCKET_GRPC "~/bucket-grpc"
    cd ~/bucket-grpc
    ~/dev/gcsfuse-tools/write-test/write_master.sh
    cd -
    umount ~/bucket-grpc && sleep 50
}


alias gcsfuse=~/go/bin/gcsfuse

# install_deps
# create_bucket
# install_go_and_add_in_path
# install_gcsfuse
# init
run


