# Python script for running resnet18 model
# Usage: python3 resnet.py

import pprint
import tempfile
import os

# from IPython import display
import matplotlib.pyplot as plt

import tensorflow as tf
import tensorflow_datasets as tfds
import time
import psutil
import tensorflow_models as tfm

# These are not in the tfm public API for v2.9. They will be available in v2.10
from official.vision.serving import export_saved_model_lib
import official.core.train_lib

os.system("sudo sh -c 'echo 3 > /proc/sys/vm/drop_caches'")

exp_config = tfm.core.exp_factory.get_exp_config('resnet_imagenet')
tfds_name = 'imagenet2012'
ds_info = tfds.builder(tfds_name ).info
print(ds_info)

# Configure model
exp_config.task.model.num_classes = 1000
exp_config.task.model.input_size = [224,224,3]#list(ds_info.features["image"].shape)
exp_config.task.model.backbone.resnet.model_id = 18

# Configure training and testing data
batch_size = 1024

exp_config.task.train_data.input_path = 'myBucket/imagenet2012-tfrecords/train/train*'
exp_config.task.train_data.tfds_split = 'train'
exp_config.task.train_data.global_batch_size = batch_size

exp_config.task.validation_data.input_path = 'myBucket/imagenet2012-tfrecords/validation/val*'
exp_config.task.validation_data.tfds_split = 'test'
exp_config.task.validation_data.global_batch_size = batch_size

logical_device_names = [logical_device.name for logical_device in tf.config.list_logical_devices()]

if 'GPU' in ''.join(logical_device_names):
  print('This may be broken in Colab.')
  device = 'GPU'
elif 'TPU' in ''.join(logical_device_names):
  print('This may be broken in Colab.')
  device = 'TPU'
else:
  print('Running on CPU is slow, so only train for a few steps.')
  device = 'CPU'

if device=='CPU':
  train_steps = 30
  exp_config.trainer.steps_per_loop = 5
else:
  train_steps=1252
  exp_config.trainer.steps_per_loop = 100

exp_config.trainer.summary_interval = 100
exp_config.trainer.checkpoint_interval = train_steps
exp_config.trainer.validation_interval = 1000
exp_config.trainer.validation_steps =  ds_info.splits['test'].num_examples // batch_size
exp_config.trainer.train_steps = train_steps
exp_config.trainer.optimizer_config.learning_rate.type = 'cosine'
exp_config.trainer.optimizer_config.learning_rate.cosine.decay_steps = train_steps
exp_config.trainer.optimizer_config.learning_rate.cosine.initial_learning_rate = 0.1
exp_config.trainer.optimizer_config.warmup.linear.warmup_steps = 100

logical_device_names = [logical_device.name for logical_device in tf.config.list_logical_devices()]

if exp_config.runtime.mixed_precision_dtype == tf.float16:
    tf.keras.mixed_precision.set_global_policy('mixed_float16')

if 'GPU' in ''.join(logical_device_names):
  distribution_strategy = tf.distribute.MirroredStrategy()
elif 'TPU' in ''.join(logical_device_names):
  tf.tpu.experimental.initialize_tpu_system()
  tpu = tf.distribute.cluster_resolver.TPUClusterResolver(tpu='/device:TPU_SYSTEM:0')
  distribution_strategy = tf.distribute.experimental.TPUStrategy(tpu)
else:
  print('Warning: this will be really slow.')
  distribution_strategy = tf.distribute.OneDeviceStrategy(logical_device_names[0])

with distribution_strategy.scope():
  model_dir = tempfile.mkdtemp()
  task = tfm.core.task_factory.get_task(exp_config.task, logging_dir=model_dir)

# Running the model for given number of epochs
model, eval_logs = tfm.core.train_lib.run_experiment(
    distribution_strategy=distribution_strategy,
    task=task,
    mode='train',
    params=exp_config,
    model_dir=model_dir,
    run_post_eval=True,
    epochs=3000,
    clear_kernel_cache=True)
