#!/usr/bin/env python3
# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import time
import psutil
import subprocess

def get_ram_info():
    """
    This method gets the RAM information of the system and returns it as a dictionary.
    """
    mem_info = psutil.virtual_memory()
    
    ram_info = {}
    ram_info['total'] = f"{mem_info.total / (1024 ** 2)} MB"
    ram_info['used'] = f"{mem_info.used / (1024 ** 2)} MB"
    ram_info['free'] = f"{mem_info.available / (1024 ** 2)} MB"
    return ram_info

def clear_kernel_cache(log):
  """
  Clears the Linux kernel cache (page cache, dentries, and inodes) H
  without invoking a shell command.
  """
  log.info(f"Before: {get_ram_info()}")
  try:
      # Open the file for writing with superuser privileges
      with open('/proc/sys/vm/drop_caches', 'w', encoding='utf-8') as f:
          f.write('1')  # Clear only data page cache not dentries.
      time.sleep(1)  # Wait for the caches to be cleared
      log.info("Kernel cache cleared successfully.")
  except (IOError, OSError) as e:
      log.error(f"Error clearing kernel cache: {e}")
      log.info(f"Falling back to clearing using shell command, for password less sudo access.")
      clear_kernel_cache_bash(log)
  log.info(f"After: {get_ram_info()}")

def clear_kernel_cache_bash(log):
    try:
        # Attempt to clear the cache with sudo, but suppress password prompt
         subprocess.run(['sudo', 'sh', '-c', 'echo 1 > /proc/sys/vm/drop_caches'], check=True, stdout=subprocess.DEVNULL, 
stderr=subprocess.DEVNULL)
         time.sleep(1)  # Wait for the caches to be cleared
    except subprocess.CalledProcessError as e:
         # If sudo fails (likely due to no passwordless access), log the error and exit
         log.warn(f"Failed to clear kernel cache: {e}")
