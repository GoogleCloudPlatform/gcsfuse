cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"

echo "Setting up the machine with Docker and Nvidia Driver"
chmod +x ml_tests/pytorch_dino_model/setup_host.sh
source ml_tests/pytorch_dino_model/setup_host.sh


cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
sudo docker build . -f perfmetrics/scripts/ml_tests/pytorch_dino_model/Dockerfile --tag pytorch-gcsfuse

mkdir container_artifacts

# TODO: Please use the KOKORO_ARTIFACTS_DIR instead of exact value.
sudo docker run --runtime=nvidia --privileged -d -v /tmpfs/src/github/gcsfuse1/container_artifacts:/pytorch_dino/run_artifacts:rw,rshared \
--shm-size=128g pytorch-gcsfuse:latest
