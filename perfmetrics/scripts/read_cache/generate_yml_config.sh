#!/bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
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

# Running this will generate a config file, and value of few cache related
# config will be populated from environment variable.

set -e

filename=config.yml
cat > $filename << EOF
write:
  create-empty-file: true
logging:
  format: text
  severity: error
cache-dir: ${CACHE_DIR:-/tmp/read_cache/}
file-cache:
  max-size-mb: ${MAX_SIZE_MB:-100}
  cache-file-for-range-read: ${CACHE_FILE_FOR_RANGE_READ-false}
metadata-cache:
  ttl-secs: ${TTL_SECS}
  stat-cache-max-size-mb: ${STAT_CACHE_MAX_SIZE_MB}
EOF
