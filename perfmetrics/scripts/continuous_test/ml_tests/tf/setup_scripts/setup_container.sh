#!/bin/bash

# Install go lang
wget -O go_tar.tar.gz https://go.dev/dl/go1.19.4.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin

# Clone the repo and build gcsfuse
git clone "https://github.com/GoogleCloudPlatform/gcsfuse.git"
cd gcsfuse
git checkout log_rotation
go build .
cd -

# Mount the bucket
echo "Mounting the bucket"
#gcsfuse/gcsfuse --implicit-dirs --max-conns-per-host 100 --disable-http2 --log-format "text" --log-file log.txt --stackdriver-export-interval 60s ml-models-data-gcsfuse myBucket > /home/output/gcsfuse.out 2> /home/output/gcsfuse.err &

# Install tensorflow model garden library
pip3 install --user tf-models-official==2.10.0

# Fail building the container image if folder.py is not at expected location.
if [ -f "/root/.local/lib/python3.7/site-packages/official/core/train_lib.py" ]; then echo "file exists"; else echo "train_lib.py file not present in expected location. Please correct the location. Exiting"; exit 1; fi
if [ -f "/root/.local/lib/python3.7/site-packages/orbit/controller.py" ]; then echo "file exists"; else echo "controller.py file not present in expected location. Please correct the location. Exiting"; exit 1; fi

# Copying tf util files from bucket
gsutil -m cp gs://gcsfuse-ml-data/tf_kokoro_test/resnet.py .
gsutil -m cp gs://gcsfuse-ml-data/tf_kokoro_test/files-modified-2.10.0/controller.py /root/.local/lib/python3.7/site-packages/official/core/
gsutil -m cp gs://gcsfuse-ml-data/tf_kokoro_test/files-modified-2.10.0/train_lib.py /root/.local/lib/python3.7/site-packages/orbit/

# Start training the model
#nohup python3 -u resnet.py > /home/output/myprogram.out 2> /home/output/myprogram.err &

# TODO cron job
