#!/bin/bash

wget -O go_tar.tar.gz https://go.dev/dl/go1.19.5.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go_tar.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Todo: please update the branch, when log-rotation changes are merged.
# Log-rotation branch will create the logs.txt file after every 6 hours.
# Hence, we need to setup the job to delete the logs file if not required.
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse
git checkout log_rotation
go build .
cd -

# Create a directory for gcsfuse logs
mkdir  run_artifacts/gcsfuse_logs

echo "Mounting GCSFuse..."
nohup /pytorch_dino/gcsfuse/gcsfuse --type-cache-ttl=1728000s \
        --stat-cache-ttl=1728000s \
        --stat-cache-capacity=1320000 \
        --stackdriver-export-interval=60s \
        --implicit-dirs \
        --disable-http2 \
        --experimental-enable-storage-client-library \
        --max-conns-per-host=100 \
        --debug_fs \
        --debug_gcs \
        --log-file run_artifacts/gcsfuse_logs/logs.txt \
        --log-format text \
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

folder_file="/opt/conda/lib/python3.8/site-packages/torchvision/datasets/folder.py"
x=$(grep -n "def pil_loader(path: str) -> Image.Image:" $folder_file | cut -f1 -d ':')
y=$(grep -n "def accimage_loader(path: str) -> Any:" $folder_file | cut -f1 -d ':')
y=$((y - 2))
lines="$x,$y"
sed -i "$lines"'d' $folder_file
sed -i "$x"'r bypassed_code.py' $folder_file

# Fix the caching issue - comes when we run the model first time with 8
# nproc_per_node - by downloading the model in single thread environment.
python -c 'import torch;torch.hub.list("facebookresearch/xcit:main")'

# Run the pytorch Dino model
# We need to run it in foreground mode to make the container running.
echo "Running the pytorch dino model..."
experiment=dino_experiment
python3 -m torch.distributed.launch \
  --nproc_per_node=8 dino/main_dino.py \
  --arch vit_small \
  --num_workers 20 \
  --data_path gcsfuse_data/imagenet/ILSVRC/Data/CLS-LOC/train/ \
  --output_dir "./run_artifacts/$experiment" \
  --norm_last_layer False \
  --use_fp16 False \
  --clip_grad 0 \
  --epochs 800 \
  --global_crops_scale 0.25 1.0 \
  --local_crops_number 10 \
  --local_crops_scale 0.05 0.25 \
  --teacher_temp 0.07 \
  --warmup_teacher_temp_epochs 30 \
  --clip_grad 0 \
  --min_lr 0.00001

echo "Pytorch DINO model completed the training successfully!"
