#!/bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PYTORCH_VESRION=$1

# Install golang
wget -O go_tar.tar.gz https://go.dev/dl/go1.21.3.linux-amd64.tar.gz -q
rm -rf /usr/local/go && tar -C /usr/local -xzf go_tar.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Clone and build the gcsfuse master branch.
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse
CGO_ENABLED=0 go build .
cd -

# Create a directory for gcsfuse logs
mkdir  run_artifacts/gcsfuse_logs

config_filename=/pytorch_dino/gcsfuse/gcsfuse-config.yaml
cat > $config_filename << EOF
metadata-cache:
  ttl-secs: 1728000
EOF
echo "Created config-file at "$config_filename

echo "Mounting GCSFuse..."
nohup /pytorch_dino/gcsfuse/gcsfuse --foreground \
        --stat-cache-capacity=1320000 \
        --stackdriver-export-interval=60s \
        --implicit-dirs \
        --max-conns-per-host=100 \
        --debug_fuse \
        --debug_gcs \
        --log-file run_artifacts/gcsfuse.log \
        --log-format text \
        --config-file $config_filename \
       gcsfuse-ml-data gcsfuse_data > "run_artifacts/gcsfuse.out" 2> "run_artifacts/gcsfuse.err" &

# Update the pytorch library code to bypass the kernel-cache
echo "Updating the pytorch library code to bypass the kernel-cache..."
echo "
def pil_loader(path: str) -> Image.Image:
    fd = os.open(path, os.O_DIRECT)
    f = os.fdopen(fd, \"rb\")
    img = Image.open(f)
    rgb_img = img.convert(\"RGB\")
    f.close()
    return rgb_img
" > bypassed_code.py

folder_file="/opt/conda/lib/python3.10/site-packages/torchvision/datasets/folder.py"
x=$(grep -n "def pil_loader(path: str) -> Image.Image:" $folder_file | cut -f1 -d ':')
y=$(grep -n "def accimage_loader(path: str) -> Any:" $folder_file | cut -f1 -d ':')
y=$((y - 2))
lines="$x,$y"
sed -i "$lines"'d' $folder_file
sed -i "$x"'r bypassed_code.py' $folder_file

# Fix the caching issue - comes when we run the model first time with 8
# nproc_per_node - by downloading the model in single thread environment.
python -c 'import torch;torch.hub.list("facebookresearch/xcit:main")'

ARTIFACTS_BUCKET_PATH="gs://gcsfuse-ml-tests-logs/ci_artifacts/pytorch/${PYTORCH_VESRION}/dino"
echo "Update status file"
echo "RUNNING" > status.txt
gsutil cp status.txt $ARTIFACTS_BUCKET_PATH/

echo "Update start time file"
echo $(date +"%s") > start_time.txt
gsutil cp start_time.txt $ARTIFACTS_BUCKET_PATH/

(
  set +e
  # Run the pytorch Dino model
  # We need to run it in foreground mode to make the container running.
  echo "Running the pytorch dino model..."
  experiment=dino_experiment
  torchrun \
    --nproc_per_node=2 dino/main_dino.py \
    --arch vit_small \
    --num_workers 20 \
    --data_path gcsfuse_data/imagenet/ILSVRC/Data/CLS-LOC/train/ \
    --output_dir "./run_artifacts/$experiment" \
    --norm_last_layer False \
    --use_fp16 False \
    --clip_grad 0 \
    --epochs 80 \
    --global_crops_scale 0.25 1.0 \
    --local_crops_number 10 \
    --local_crops_scale 0.05 0.25 \
    --teacher_temp 0.07 \
    --warmup_teacher_temp_epochs 30 \
    --clip_grad 0 \
    --min_lr 0.00001
    if [ $? -eq 0 ];
    then
        echo "Pytorch dino model completed the training successfully!"
        echo "COMPLETE" > status.txt
    else
        echo "Pytorch dino model training failed!"
        echo "ERROR" > status.txt
    fi
)

gsutil cp status.txt $ARTIFACTS_BUCKET_PATH/
