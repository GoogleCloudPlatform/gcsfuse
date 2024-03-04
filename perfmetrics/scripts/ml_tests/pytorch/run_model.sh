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
NUM_EPOCHS=80
TEST_BUCKET="gcsfuse-ml-data"

# Install golang
wget -O go_tar.tar.gz https://go.dev/dl/go1.22.0.linux-amd64.tar.gz -q
rm -rf /usr/local/go && tar -C /usr/local -xzf go_tar.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Clone and build the gcsfuse master branch.
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse
CGO_ENABLED=0 go build .
cd -

# Create a directory for gcsfuse logs
mkdir  run_artifacts/gcsfuse_logs

# We have created a bucket in the asia-northeast1 region to align with the location of our PyTorch 2.0 VM, which is also in asia-northeast1.
if [ ${PYTORCH_VESRION} == "v2" ];
then
  TEST_BUCKET="gcsfuse-ml-data-asia-northeast1"
fi

config_filename=/tmp/gcsfuse_config.yaml
cat > $config_filename << EOF
logging:
  file-path: run_artifacts/gcsfuse.log
  format: text
  severity: trace
  log-rotate:
    max-file-size-mb: 1024
    backup-file-count: 3
    compress: true
metadata-cache:
  ttl-secs: 1728000
  stat-cache-max-size-mb: 3200
EOF
echo "Created config-file at "$config_filename

echo "Mounting GCSFuse..."
nohup /pytorch_dino/gcsfuse/gcsfuse --foreground \
        --stackdriver-export-interval=60s \
        --implicit-dirs \
        --max-conns-per-host=100 \
        --config-file $config_filename \
      $TEST_BUCKET gcsfuse_data > "run_artifacts/gcsfuse.out" 2> "run_artifacts/gcsfuse.err" &

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

# (TulsiShah) TODO: Pytorch 2.0 compile mode has issues (https://github.com/pytorch/pytorch/issues/94599),
# which is fixed in pytorch version 2.1.0 (https://github.com/pytorch/pytorch/pull/100071).
# We'll remove this workaround once we update our Docker image to use Pytorch 2.1.0 or greater version.
if [ ${PYTORCH_VESRION} == "v2" ];
then
  allowed_functions_file="/opt/conda/lib/python3.10/site-packages/torch/_dynamo/allowed_functions.py"
  # Update the pytorch library code to bypass the kernel-cache
  echo "Updating the pytorch library code to Disallow_in_graph distributed API.."
  echo "
def _disallowed_function_ids():
  remove = [
      True,
      False,
      None,
      collections.OrderedDict,
      copy.copy,
      copy.deepcopy,
      inspect.signature,
      math.__package__,
      torch.__builtins__,
      torch.autocast_decrement_nesting,
      torch.autocast_increment_nesting,
      torch.autograd.grad,
      torch.clear_autocast_cache,
      torch.cuda.current_device,
      torch.cuda.amp.autocast_mode.autocast,
      torch.cpu.amp.autocast_mode.autocast,
      torch.distributions.constraints.is_dependent,
      torch.distributions.normal.Normal,
      torch.inference_mode,
      torch.set_anomaly_enabled,
      torch.set_autocast_cache_enabled,
      torch.set_autocast_cpu_dtype,
      torch.set_autocast_cpu_enabled,
      torch.set_autocast_enabled,
      torch.set_autocast_gpu_dtype,
      torch.autograd.profiler.profile,
      warnings.warn,
      torch._C._dynamo.eval_frame.unsupported,
  ]
  # extract all dtypes from torch
  dtypes = [
      obj for obj in torch.__dict__.values() if isinstance(obj, type(torch.float32))
  ]
  remove += dtypes
  storage = [
      obj
      for obj in torch.__dict__.values()
      if isinstance(obj, type(torch.FloatStorage))
  ]
  remove += storage

  # Distributed APIs don't work well with torch.compile.
  if torch.distributed.is_available():
      remove.extend(
           torch.distributed.distributed_c10d.dynamo_unsupported_distributed_c10d_ops
      )

  return {id(x) for x in remove}
" > disallowed_function.py

  x=$(grep -n "def _disallowed_function_ids():" $allowed_functions_file | cut -f1 -d ':')
  y=$(grep -n "def _allowed_function_ids():" $allowed_functions_file | cut -f1 -d ':')
  y=$((y - 3))
  lines="$x,$y"
  sed -i "$lines"'d' $allowed_functions_file
  sed -i "$x"'r disallowed_function.py' $allowed_functions_file

  distributed_c10d_file="/opt/conda/lib/python3.10/site-packages/torch/distributed/distributed_c10d.py"
  echo "# This ops are not friendly to TorchDynamo. So, we decide to disallow these ops
# in FX graph, allowing them to run them on eager, with torch.compile.
dynamo_unsupported_distributed_c10d_ops = [
      all_reduce_multigpu,
      recv,
      all_gather_object,
      all_gather_coalesced,
      all_to_all_single,
      all_reduce,
      gather_object,
      all_to_all,
      all_reduce_coalesced,
      gather,
      broadcast_object_list,
      barrier,
      reduce_multigpu,
      scatter,
      scatter_object_list,
      reduce,
      reduce_scatter_multigpu,
      all_gather,
      broadcast_multigpu,
      all_gather_multigpu,
      reduce_scatter,
      all_gather_into_tensor,
      broadcast,
      reduce_scatter_tensor,
      send,
]" >> $distributed_c10d_file
fi

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
    --epochs $NUM_EPOCHS \
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
