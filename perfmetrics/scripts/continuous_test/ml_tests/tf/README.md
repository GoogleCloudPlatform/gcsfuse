# Kokoro test for tf model reliability using gcsfuse

This readme contains file descriptions, libraries and model used for the test.

## Test description

* We have used tf2.10 framework along with tf-model-garden library (v2.10) 
for [resnet18](https://www.tensorflow.org/tfmodels/vision/image_classification) based reliability testing on ImageNet dataset for 3000 epochs
which runs for roughly 14 days.

## Packages required

* The [TensorFlow Model Garden](https://github.com/tensorflow/models) is a repository with a number of different 
implementations of state-of-the-art (SOTA) models and modeling solutions for TensorFlow users.

* [Docker engine](https://docs.docker.com/engine/install/ubuntu/) for running deep learning container based image.

* [Nvidia drivers](https://docs.nvidia.com/datacenter/tesla/tesla-installation-notes/index.html#runfile) for using gpu.
We have used driver version 450.172.01 for experiments.

## File description

* build.sh: Entrypoint for kokoro vm. Runs setup_host.sh for installing required nvidia drivers
and docker engine. And starts experiment using setup_scripts/Dockerfile as container image

* Dockerfile: Uses Deep learning container tf as a base image.

* setup_container.sh: Entrypoint for the Docker container. Installs gcsfuse and tf-model-garden
library and starts the experiment in the container

* resnet_runner.py: python script for running resnet18 model using tf-model-garden library.

## Config changes for running model

In resnet_runner.py, batch_size can be adjusted on line 34 and number of epochs for the training can be specified in call
to tfm.core.train_lib.run_experiment at line 100

## Logging
4 hours of GCSFuse logs with debug flags: --debug_fuse, --debug_gcs take around 32 GiB of space on disk. With our 
disk space for kokor vm (1000gb), we store logs only for last 72 hours.
The gcsfuse based logs are stored in directory ${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/container_artifacts/logs
while the gcsfuse errors and output (Mounted successfully) are stored in ${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/container_artifacts/output
