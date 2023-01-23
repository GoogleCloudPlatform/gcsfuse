# Execution of Pytorch DINO Model

As an automation, we run the pytorch dino model in a docker container. In docker,
we use word host to specify the actual VM and container to specify running
docker image. Please find the description of all involved scripts in this 
automation with their purpose:

## File Descriptions:

### File: perfmetrics/scripts/ml_test/setup_host.sh
By executing this script, we setup the host machine by installing the ops-agent,
docker system, nvidia-driver (gpu based ml training), and some utilities like,
curl, ca-certificates, lsb-release etc.

### File: perfmetrics/scripts/ml_test/pytorch/dino/setup_container.sh
This script contains the instruction to install gcsfuse, mount GCS-bucket
using gcsfuse, and finally runs the pytorch dino model.

### File: perfmetrics/scripts/continuous_test/pytorch/dino/build.sh
This is the parent script of the above two scripts. Firstly, it sets-up the host
machine after that it creates the docker-image and finally it runs the container
with the inststructions written in the setup_container.sh.

## Artifacts after the Executions:
After the execution of kokoro job, we copy two types of logs -
(a) GCSFuse logs
(b) Dino model logs.

### GCSFuse Logs: container_artifacts/gcsfuse_logs
We mount the gcsfuse with debug flags, this folder contains the running gcsfuse
logs. This will be beneficial for debugging purpose.

### Dino Model Logs: container_artifacts/dino-experiment/
checkpoint*.pth - Model checkpointing. 
log.txt - Contains the model learning parameter value after each epoch.

### Steps to run the model on VM 
1. Create an A2 GPU instance with 8 GPU on GCP console.
2. Create a Working directory, and sets the KOKORO_ARTIFACTS_DIR environment 
variable - with current working directory.
3. Create a folder named "github" and clone the gcsfuse repo in that.
4. Run the below script in the current working directory:
   **source github/gcsfuse/permetrics/scripts/continuous_test/ml_tests/pytorch/dino/build.sh**
5. The above command first setups the host and then start running the model
inside container.
