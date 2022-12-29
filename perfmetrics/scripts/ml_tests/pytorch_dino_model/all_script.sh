git clone https://github.com/facebookresearch/dino.git

# Install gcsfuse
# export GCSFUSE_REPO=gcsfuse-`lsb_release -c -s`
# echo "deb https://packages.cloud.google.com/apt $GCSFUSE_REPO main" | sudo tee /etc/apt/sources.list.d/gcsfuse.list
# curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -

# sudo apt-get update
# sudo apt-get install gcsfuse


# Create the mount directory
mkdir gcsfuse_data

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

#Install pytorch dependency
pip3 install timm

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

# Install the go lang and build the 
wget https://go.dev/dl/go1.19.4.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.19.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

git clone https://github.com/raj-prince/gcsfuse.git
cd gcsfuse
git checkout log_rotation_latest
go build .


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



# Steps to bootstrap with gpu docker

# Install docker
sudo apt-get update
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release


sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null


  sudo apt-get update

  sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin -y

#Install driver
sudo apt update && sudo apt install -y build-essential
BASE_URL=https://us.download.nvidia.com/tesla
DRIVER_VERSION=450.172.01
curl -fSsl -O $BASE_URL/$DRIVER_VERSION/NVIDIA-Linux-x86_64-$DRIVER_VERSION.run

sudo sh NVIDIA-Linux-x86_64-$DRIVER_VERSION.run -s

#Install Nvidia container tool
distribution=$(. /etc/os-release;echo $ID$VERSION_ID) \
      && curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
      && curl -s -L https://nvidia.github.io/libnvidia-container/$distribution/libnvidia-container.list | \
            sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
            sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list

sudo apt-get update
sudo apt-get install -y nvidia-docker2
sudo systemctl restart docker