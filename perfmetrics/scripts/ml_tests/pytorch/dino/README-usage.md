# Execution of Pytorch DINO Model

As an automation, we run the pytorch dino model as a docker container. In docker,
we say use word host to specify the actual VM and container to specify running
docker image. Please find the description of all involved scripts in this 
automation with their purpose:

## File Descriptions:

### File: perfmetrics/scripts/ml_test/setup_host.sh
By executing this script, we setup the host machine by installing the ops-agent,
docker system, nvidia-driver (gpu based ml training), and some utilities like,
curl, ca-certificates, lsb-release etc.

### File: perfmetrics/scripts/ml_test/pytorch/dino/setup_container.sh
This shell scripts contains the instruction to install gcsfuse, mount GCS-bucket
using gcsfuse, some log-related handlings and finally runs the pytorch dino model.

### File: perfmetrics/scripts/continuous_test/pytorch/dino/build.sh
This is the parent script of the above two scripts. Firstly, it sets-up the host
machine after that it creats the docker-image and finally it runs the container
with the inststructions written in the setup_container.sh.

## Artifacts after the Executions:
As a kokoro build execution, we create various logs - (a) GCSFuse logs (b) Dino
model logs.

### GCSFuse Logs: container_artifacts/gcsfuse_logs
We mount the gcsfuse with debug flags, this folder contains the running gcsfuse
logs. This will be beneficial for debugging purpose.

### Dino Model Logs: container_artifacts/dino-experiment/
checkpoint*.pth - Model checkpointing. 
log.txt - Contains the standard ouput we get after execution of DINO model.




