#!/bin/bash
set -e
# Refer for env var: https://stackoverflow.com/questions/72441758/typeerror-descriptors-cannot-not-be-created-directly
export PROTOCOL_BUFFERS_PYTHON_IMPLEMENTATION=python
echo Installing pip and fuse..
sudo apt-get install fuse -y
sudo apt-get install pip -y
echo Installing requirements..
pip install --require-hashes -r requirements.txt --user

GCSFUSE_FLAGS=$1
UPLOAD_FLAGS=$2

echo Running script..
# TODO (ruchikasharmaa): Changed name of bucket in config.json file to 'list-benchmark-test1' for running periodic experiments.
#  The bucket is in gcs-fuse-test project. Before merging to master, we need to resolve the conflicts.
python3 listing_benchmark.py config.json --gcsfuse_flags "$GCSFUSE_FLAGS" $UPLOAD_FLAGS --command "ls -R" --num_samples 30 --message "Testing CT setup."
