git clone https://github.com/facebookresearch/dino.git

# Install gcsfuse
export GCSFUSE_REPO=gcsfuse-`lsb_release -c -s`
echo "deb https://packages.cloud.google.com/apt $GCSFUSE_REPO main" | sudo tee /etc/apt/sources.list.d/gcsfuse.list
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -

sudo apt-get update
sudo apt-get install gcsfuse


# Create the mount directory
mkdir gcsfuse_data

# Mount gcsfuse
nohup gcsfuse --type-cache-ttl=100000s \
        --stat-cache-ttl=100000s \
        --stat-cache-capacity=1320000 \
        --stackdriver-export-interval=60s \
        --implicit-dirs \
        --disable-http2 \
        --max-conns-per-host=100 \
       gcsfuse-ml-data gcsfuse_data > gcsfuse.out 2> gcsfuse.err &

#Install pytorch dependency
pip3 install timm

# Run the pytorch Dino model
experiment=dino_experiment
nohup python3 -m torch.distributed.launch \
  --nproc_per_node=1 dino/main_dino.py \
  --arch vit_small \
  --num_workers 20 \
  --data_path gcsfuse_data/imagenet/ILSVRC/Data/CLS-LOC/train/ \
  --output_dir ./$experiment \
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
  --min_lr 0.00001 > $experiment.out 2> $experiment.err &



