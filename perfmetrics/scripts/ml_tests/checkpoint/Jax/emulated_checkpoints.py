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

import jax
import jax.numpy as jnp
import optax
from flax import linen as nn
from flax.training import train_state
from flax.training import checkpoints
import argparse

# Mock model definition.
class SimpleModel(nn.Module):
  @nn.compact
  def __call__(self, x):
    features = [16384, 8192, 4096, 2048, 1024, 512, 256, 128, 1]
    for feature in features:
      x = nn.Dense(features=feature)(x)
      x = nn.relu(x)
    return x

# Mock training step.
def train_step(state, batch):
  def loss_fn(params):
    preds = state.apply_fn(params, batch['x'])
    loss = jnp.mean(jnp.square(preds - batch['y']))
    return loss

  grad_fn = jax.value_and_grad(loss_fn)
  loss, grads = grad_fn(state.params)
  state = state.apply_gradients(grads=grads)
  return state, loss

if __name__ == "__main__":
  parser = argparse.ArgumentParser(description='Train a simple model and save checkpoints.')
  parser.add_argument('--checkpoint_dir', type=str, required=True, help='Directory to save checkpoints')
  parser.add_argument('--num_train_steps', type=int, default=2000, help='Number of training steps') # Added argument for num_train_steps
  args = parser.parse_args()

  # Sample data.
  key = jax.random.PRNGKey(0)
  x = jax.random.normal(key, (10, 5))
  y = jax.random.normal(key, (10, 1))

  # Initialize model and optimizer.
  model = SimpleModel()
  params = model.init(key, x)
  optimizer = optax.adam(learning_rate=0.01)

  # Create train state.
  state = train_state.TrainState.create(apply_fn=model.apply, params=params, tx=optimizer)

  # Mock training step.
  state, loss = train_step(state, {'x': x, 'y': y})

  # Save checkpoint to local directory
  for step in range(args.num_train_steps):
    if step % 200 == 0:
      checkpoints.save_checkpoint(args.checkpoint_dir, state, step, keep=100, prefix='checkpoint_')
