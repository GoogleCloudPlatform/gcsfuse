This readme contains the code flow for tf based kokoro job.

* The Kokoro vm will start with build.sh, setting up the vm with nvidia drivers 
and docker using gcsfuse/perfmetrics/scripts/ml_tests/setup_host.sh. 

* We have used dl-tf2.10 based deep learning container as our base image for docker.
* setup_container.sh does go installation, then clone and build gcsfuse from log_rotation branch in the gcsfuse repo.

* Then we install tf-models-official v2.10.0 and make necessary changes in 
/root/.local/lib/python3.7/site-packages/official/core/train_lib.py and
/root/.local/lib/python3.7/site-packages/orbit/controller.py files for implementing 
epochs and clear_kernel_cache features.

* resnet_runner.py contains the necessary details for setting up resnet18 model and 
creating a tf.dataset instance and training the model on the data for epochs specified.
