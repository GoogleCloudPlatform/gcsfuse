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
# Upload data to the gsheet only when it runs through kokoro.
if [ "${KOKORO_JOB_TYPE}" != "RELEASE" ] && [ "${KOKORO_JOB_TYPE}" != "CONTINUOUS_INTEGRATION" ] && [ "${KOKORO_JOB_TYPE}" != "PRESUBMIT_GITHUB" ];
then
  python3 listing_benchmark.py config.json --command "ls -R" --num_samples 30 --message "Testing CT setup."
else
  python3 listing_benchmark.py config.json --command "ls -R" --num_samples 30 --upload --message "Testing CT setup."
fi