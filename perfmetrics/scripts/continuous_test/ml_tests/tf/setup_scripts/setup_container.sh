#!/bin/bash

# Installs go1.19 on the container, builds gcsfuse using log_rotation file
# and installs tf-models-official v2.10.0, makes update to include clear_kernel_cache
# and epochs functionality, and runs the model

# Install go lang
wget -O go_tar.tar.gz https://go.dev/dl/go1.19.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin

# Clone the repo and build gcsfuse
# TODO change the branch once log_rotation is merged
git clone "https://github.com/GoogleCloudPlatform/gcsfuse.git"
cd gcsfuse
git checkout log_rotation
go build .
cd -

# Mount the bucket and run in background so that docker doesn't keep running after resnet_runner.py fails
echo "Mounting the bucket"
nohup gcsfuse/gcsfuse --implicit-dirs --experimental-enable-storage-client-library --debug_fuse --debug_gcs --max-conns-per-host 100 --disable-http2 --log-format "text" --log-file /home/logs/log.txt --stackdriver-export-interval 60s ml-models-data-gcsfuse myBucket > /home/output/gcsfuse.out 2> /home/output/gcsfuse.err &

# Install tensorflow model garden library
pip3 install --user tf-models-official==2.10.0

# Fail building the container image if train_lib.py and controller.py are not at expected location.
if [ -f "/root/.local/lib/python3.7/site-packages/official/core/train_lib.py" ]; then echo "file exists"; else echo "train_lib.py file not present in expected location. Please correct the location. Exiting"; exit 1; fi
if [ -f "/root/.local/lib/python3.7/site-packages/orbit/controller.py" ]; then echo "file exists"; else echo "controller.py file not present in expected location. Please correct the location. Exiting"; exit 1; fi

# Adding cache clearing functionality and epochs in controller.py
echo "
  def train(self, steps: int, checkpoint_at_completion: bool = True, epochs = 1, clear_kernel_cache = False):
    \"\"\"Runs training until the specified global step count has been reached.

    This method makes calls to \`self.trainer.train()\` until the global step
    count is equal to \`steps\`. It will additionally save checkpoints (if a
    \`CheckpointManager\` was passed to \`Controller.__init__\`) and summarize
    training output (if \`summary_dir\` is set).

    Args:
      steps: The global step count to train up to.
      checkpoint_at_completion: Whether to save a checkpoint when this method
        returns (regardless of the checkpointing interval). Defaults to \`True\`.
    \"\"\"
    self._require(\"trainer\", for_method=\"train\")
    total_steps = steps
    for _ in range(epochs):
      # TODO(momernick): Support steps=None or -1 (training to exhaustion).
      current_step = self.global_step.numpy()  # Cache, since this is expensive.
      _log(f\"train | step: {current_step: 6d} | training until step {steps}...\")
      while current_step < total_steps:
        # Calculates steps to run for the next train loop.
        num_steps = min(total_steps - current_step, self.steps_per_loop)
        self._train_n_steps(num_steps)
        self._maybe_save_checkpoint()
        current_step = self.global_step.numpy()
      total_steps += steps

      if clear_kernel_cache:
        os.system(\"sudo sh -c 'echo 3 > /proc/sys/vm/drop_caches'\")

    if checkpoint_at_completion:
      self._maybe_save_checkpoint(check_interval=False)
" > bypassed_code.py

controller_file="/root/.local/lib/python3.7/site-packages/orbit/controller.py"
x=$(grep -n "def train(self, steps: int, checkpoint_at_completion: bool = True):" $controller_file | cut -f1 -d ':')
y=$(grep -n "def evaluate(self, steps: int = -1)" $controller_file | cut -f1 -d ':')
y=$((y - 2))
lines="$x,$y"
sed -i "$lines"'d' $controller_file
sed -i "$x"'r bypassed_code.py' $controller_file

echo "
import os
import time
" > bypassed_code.py

x=$(grep -n "import time" $controller_file | cut -f1 -d ':')
lines="$x,$x"
sed -i "$lines"'d' $controller_file
sed -i "$x"'r bypassed_code.py' $controller_file

# Adding params for clear_kernel_cache and epochs in train_lib.py
echo "
def run_experiment(
  distribution_strategy: tf.distribute.Strategy,
  task: base_task.Task,
  mode: str,
  params: config_definitions.ExperimentConfig,
  model_dir: str,
  run_post_eval: bool = False,
  save_summary: bool = True,
  train_actions: Optional[List[orbit.Action]] = None,
  eval_actions: Optional[List[orbit.Action]] = None,
  trainer: Optional[base_trainer.Trainer] = None,
  controller_cls=orbit.Controller,
  epochs: int = 1,
  clear_kernel_cache: bool = False
) -> Tuple[tf.keras.Model, Mapping[str, Any]]:
  \"\"\"Runs train/eval configured by the experiment params.

  Args:
    distribution_strategy: A distribution distribution_strategy.
    task: A Task instance.
    mode: A 'str', specifying the mode. Can be 'train', 'eval', 'train_and_eval'
      or 'continuous_eval'.
    params: ExperimentConfig instance.
    model_dir: A 'str', a path to store model checkpoints and summaries.
    run_post_eval: Whether to run post eval once after training, metrics logs
      are returned.
    save_summary: Whether to save train and validation summary.
    train_actions: Optional list of Orbit train actions.
    eval_actions: Optional list of Orbit eval actions.
    trainer: the base_trainer.Trainer instance. It should be created within the
      strategy.scope().
    controller_cls: The controller class to manage the train and eval process.
      Must be a orbit.Controller subclass.

  Returns:
    A 2-tuple of (model, eval_logs).
      model: \`tf.keras.Model\` instance.
      eval_logs: returns eval metrics logs when run_post_eval is set to True,
        otherwise, returns {}.
  \"\"\"
  runner = OrbitExperimentRunner(
      distribution_strategy=distribution_strategy,
      task=task,
      mode=mode,
      params=params,
      model_dir=model_dir,
      run_post_eval=run_post_eval,
      save_summary=save_summary,
      train_actions=train_actions,
      eval_actions=eval_actions,
      trainer=trainer,
      controller_cls=controller_cls,
  )
  return runner.run(epochs=epochs, clear_kernel_cache=clear_kernel_cache)
" > bypassed_code.py

train_lib_file="/root/.local/lib/python3.7/site-packages/official/core/train_lib.py"
x=$(grep -n "def run_experiment(" $train_lib_file | cut -f1 -d ':')
y=$(grep -n "return runner.run()" $train_lib_file | cut -f1 -d ':')
lines="$x,$y"
sed -i "$lines"'d' $train_lib_file
x=$((x-1))
sed -i "$x"'r bypassed_code.py' $train_lib_file

echo "  def run(self, epochs=1, clear_kernel_cache=False) -> Tuple[tf.keras.Model, Mapping[str, Any]]:" > bypassed_code.py
x=$(grep -n "def run(self) -> Tuple\[tf.keras.Model, Mapping\[str, Any\]\]:" $train_lib_file | cut -f1 -d ':')
lines="$x,$x"
sed -i "$lines"'d' $train_lib_file
x=$((x-1))
sed -i "$x"'r bypassed_code.py' $train_lib_file

echo "
      if mode == 'train' or mode == 'train_and_post_eval':
        self.controller.train(steps=params.trainer.train_steps, clear_kernel_cache=clear_kernel_cache, epochs=epochs)" > bypassed_code.py
x=$(grep -n "if mode == 'train' or mode == 'train_and_post_eval':" $train_lib_file | cut -f1 -d ':')
y=$(grep -n "self.controller.train(steps=params.trainer.train_steps)" $train_lib_file | cut -f1 -d ':')
lines="$x,$y"
sed -i "$lines"'d' $train_lib_file
x=$((x-1))
sed -i "$x"'r bypassed_code.py' $train_lib_file

# Start training the model
python3 -u resnet_runner.py
