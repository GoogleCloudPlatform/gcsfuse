# Install the go lang and build the 
wget https://go.dev/dl/go1.19.4.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.19.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

git clone https://github.com/raj-prince/gcsfuse.git
cd gcsfuse
git checkout log_rotation_latest
go build .

# Mount gcsfuse
nohup /home/princer_google_com/gcsfuse/gcsfuse --type-cache-ttl=1728000s \
        --stat-cache-ttl=1728000s \
        --stat-cache-capacity=1320000 \
        --stackdriver-export-interval=60s \
        --implicit-dirs \
        --disable-http2 \
        --max-conns-per-host=100 \
        --debug_fs \
        --debug_gcs \
        --log-file logs.txt \
        --log-format text \
       gcsfuse-ml-data gcsfuse_data > gcsfuse.out 2> gcsfuse.err &

# Update the pytorch library code to bypass the kernel-cache
echo "
def pil_loader(path: str) -> Image.Image:
    fd = os.open(path, os.O_DIRECT)
    f = os.fdopen(fd, \"rb\")
    img = Image.open(f)
    rgb_img = img.convert(\"RGB\")
    f.close()
    return rgb_img
" > bypassed_code.py

folder_file="/usr/local/lib/python3.6/dist-packages/torchvision/datasets/folder.py"
x=$(grep -n "def pil_loader(path: str) -> Image.Image:" $folder_file | cut -f1 -d ':')
incr=4
y=$((x + incr))
lines="$x,$y"
sed -i "$lines"'d' $folder_file
sed -i "$x"'r bypassed_code.py' $folder_file

# Fix the caching issue, by downloading the issue
python -c "import torch;torch.hub.list("facebookresearch/xcit:main")"

# Run the pytorch Dino model
experiment=dino_experiment
nohup python3 -m torch.distributed.launch \
  --nproc_per_node=8 dino/main_dino.py \
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



# Setup log-deleter - although we should not delete any logs - it should be archived.
echo "
num_logs=`ls logs* | wc -w`

if [ $num_logs -lt 3 ]
then
        exit 0
fi

logs_list=`ls -tr logs*`

for log_file in $logs_list; do
        ((num_logs--))
        `rm $log_file`

        if [ $num_logs -lt 3 ]
        then
                exit 0
        fi
done
" > log_deleter.sh
chmod +x log_deleter.sh

# Cron job setup to execute the log-deleter script periodically
#TODO