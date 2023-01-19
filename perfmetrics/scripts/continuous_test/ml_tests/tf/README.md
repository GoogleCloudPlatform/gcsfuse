# Kokoro test for tf model reliability testing

This readme contains file descriptions, libraries and model used for the test.

## Test description:

* We have used tf2.10 framework along with tf-model-garden library (v2.10) 
for resnet18 based reliability testing on ImageNet dataset for 2000 epochs
which runs for roughly 11.5 days.

* The TensorFlow Model Garden is a repository with a number of different 
implementations of state-of-the-art (SOTA) models and modeling solutions for TensorFlow users.

* build.sh: Entrypoint for kokoro vm. Runs setup_host.sh for installing required nvidia drivers
and docker engine. And starts experiment using setup_scripts/Dockerfile as container image

* Dockerfile: Uses Deep learning container tf as a base image.

* setup_container.sh: Entrypoint for the Docker container. Installs gcsfuse and tf-model-garden
library and starts the experiment on the container

* resnet_runner.py: python script for running resnet18 model using tf-model-garden library.
Batch_size can be adjusted on line 34 and number of epochs for the training can be specified in call
to tfm.core.train_lib.run_experiment at line 100

## Logging
The gcsfuse based logs are stored in directory ${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/container_artifacts/logs
while the gcsfuse errors and output are stored in ${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/container_artifacts/output
