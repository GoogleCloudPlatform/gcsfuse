#!/bin/bash
set -e
# Refer for env var: https://stackoverflow.com/questions/72441758/typeerror-descriptors-cannot-not-be-created-directly
export PROTOCOL_BUFFERS_PYTHON_IMPLEMENTATION=python
echo Installing pip and fuse..
sudo apt-get install fuse -y
sudo apt-get install pip -y
echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
echo Running script..
GCSFUSE_FLAGS=$1
UPLOAD_FLAGS=$2
python3 listing_benchmark.py config.json --gcsfuse_flags "$GCSFUSE_FLAGS" $UPLOAD_FLAGS --command "ls -R" --num_samples 30 --message "Testing CT setup."
