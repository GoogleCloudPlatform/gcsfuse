#i!/bin/bash
set -e
sudo apt-get update

echo "Installing Golang"
sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt-get update
sudo apt-get install golang-go -y
echo 'export GOROOT=/usr/lib/go' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOROOT/bin:$GOPATH/bin' >> ~/.bashrc 
source ~/.bashrc

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"

echo "Setting up a machine"
chmod +x ml_tests/setup.sh
source ml_tests/setup.sh

echo "Running ML model automation script"
cd ml_tests
python3 run_image_recognition_models.py -- fashion_items_image_recognition_model/fashion_items_image_recognition_model.py fashion_items_image_recognition_model/requirements.txt --data_read_method gcsfuse --gcsbucket_data_path fashion_items_data_small/data test
