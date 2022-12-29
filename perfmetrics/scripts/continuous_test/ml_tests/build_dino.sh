cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"

echo "Setting up the machine with Docker and Nvidia Driver"
chmod +x ml_tests/pytorch_dino_model/host_setup.sh
source ml_tests/pytorch_dino_model/host_setup.sh


cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse
docker build . -f perfmetrics/ml_tests/pytorch_dino_model/Dockerfile -t dlc-gcsfuse
