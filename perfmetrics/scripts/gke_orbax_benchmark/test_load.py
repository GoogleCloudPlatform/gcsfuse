# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# test_load.py

import os
import re
import time
from absl import app
from absl import logging
import click
from etils import epath
import jax
import numpy as np
import orbax.checkpoint as ocp


def set_no_of_jax_cpu(num_cpu_devices):
  # This session needs to be run before any JAX code.
  jax.config.update('jax_platforms', 'cpu')
  xla_flags = os.getenv('XLA_FLAGS', '')
  xla_flags = re.sub(
      r'--xla_force_host_platform_device_count=\S+', '', xla_flags
  ).split()
  os.environ['XLA_FLAGS'] = ' '.join(
      [f'--xla_force_host_platform_device_count={num_cpu_devices}'] + xla_flags
  )


def load_ckpt(path, backend):
  def update_devices(x):
    # update the sharding of the metadata to load into CPU devices
    if isinstance(x, ocp.metadata.ArrayMetadata):
      if isinstance(x.sharding, ocp.metadata.NamedShardingMetadata):
        if backend in ('cpu', 'tpu'):
          mesh = jax.sharding.Mesh(
              np.asarray(jax.devices(backend=backend)), x.sharding.axis_names
          )
          pspec = jax.sharding.PartitionSpec(*x.sharding.partition_spec)
          sharding = jax.sharding.NamedSharding(mesh, pspec)
          x.sharding = ocp.metadata.NamedShardingMetadata.from_jax_sharding(
              sharding
          )
        else:
          x.sharding = None
      else:
        if backend in ('cpu', 'tpu'):
          # assume sharding
          if (
              len(x.shape)
              and x.shape[0] % jax.device_count(backend=backend) != 0
          ):
            raise ValueError(f'Unable to shard shape={x.shape}')

          mesh = jax.sharding.Mesh(
              jax.devices(backend=backend), axis_names=('x',)
          )
          if len(x.shape):
            pspec = jax.sharding.PartitionSpec('x')
          else:
            pspec = jax.sharding.PartitionSpec()
          sharding = jax.sharding.NamedSharding(mesh, pspec)
          x.sharding = ocp.metadata.NamedShardingMetadata.from_jax_sharding(
              sharding
          )
    return x

  with ocp.StandardCheckpointer(restore_concurrent_gb= int(os.environ['RESTORE_CONCURRENT_GB']) if 'RESTORE_CONCURRENT_GB' in os.environ else 512) as ckptr:
    metadata = ckptr.metadata(path)
    items = jax.tree.map(update_devices, metadata)
    items = jax.tree.map(ocp.utils.to_shape_dtype_struct, items)

    def restore():
      return ckptr.restore(path, target=items)

    stime = time.time()
    restored = restore()
    etime = time.time()
    elapsed = etime - stime

    restored_types = set()
    jax.tree.map(lambda x: restored_types.add(type(x)), restored)
    logging.info('restored_types = %s', restored_types)

    return elapsed, restored


def save_ckpt(item, path):
  with ocp.StandardCheckpointer() as ckptr:
    ckptr.save(path, state=item)
  saved_sizes = []
  jax.tree.map(
      lambda x: saved_sizes.append(np.prod(x.shape)) * x.dtype.itemsize, item
  )
  logging.info(f'Saved {np.sum(saved_sizes) / (1000**3)} GB')


@click.group()
def cli():
  pass


@click.command()
@click.option('--path', type=str, default=None, help='path to load')
@click.option(
    '--backend',
    default='cpu',
    type=click.Choice(['cpu', 'tpu', 'numpy']),
    help='backend to use',
)
@click.option('--num', default=5, help='Number of iterations.')
@click.option('--cpuno', default=4, help='Number of cpus.')
def load_test(path, backend, num, cpuno):
  """Load the checkpoint and time it."""
  if backend == 'cpu':
    set_no_of_jax_cpu(cpuno)
  if backend in ('cpu', 'tpu'):
    logging.info('Loading with %s', jax.devices(backend=backend))
    jax.clear_caches()
  elapsed_times = []
  for i in range(num):
    elapsed, _ = load_ckpt(path, backend)
    print(f'Loop_{i}: Took {elapsed}s to load')
    elapsed_times.append(elapsed)

  print(f'Average elapsed time: {np.mean(elapsed_times)}s')


@click.command()
@click.option('--path', type=str, default=None, help='input path to load')
@click.option('--output', type=str, default=None, help='output path to save')
@click.option(
    '--backend',
    default='cpu',
    type=click.Choice(['cpu', 'tpu', 'numpy']),
    help='backend to use',
)
@click.option('--cpuno', default=4, help='Number of cpus.')
def resave(path, output, backend, cpuno):
  """Load the checkpoint from `path` and save it to `output`."""
  if backend == 'cpu':
    set_no_of_jax_cpu(cpuno)
  if backend in ('cpu', 'tpu'):
    logging.info('Loading with %s', jax.devices(backend=backend))
    jax.clear_caches()
  elapsed, restored = load_ckpt(path, backend)
  print(f'Took {elapsed}s to load')

  save_ckpt(restored, output)


cli.add_command(load_test)
cli.add_command(resave)

if __name__ == '__main__':
  logging.set_verbosity(logging.INFO)
  cli()