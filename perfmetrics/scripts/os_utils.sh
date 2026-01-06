#!/bin/bash
# Copyright 2026 Google LLC
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

# PREVENT MULTIPLE SOURCING
if [ "${_OS_UTILS_SH_LOADED:-}" = "true" ]; then
  return 0
fi

_OS_UTILS_SH_LOADED=true

# Detect OS ID from /etc/os-release
get_os_id() {
  if [ -f /etc/os-release ]; then
    ( . /etc/os-release && echo "$ID" )
  else
    echo "Error: /etc/os-release not found. Cannot detect OS."
    return 1
  fi
}

# Detect and map system architecture to Go architecture
get_go_arch() {
  local system_arch=$(uname -m)
  case "$system_arch" in
    x86_64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) echo "unsupported" ;;
  esac
}

# Install packages based on OS ID
install_packages_by_os() {
  local os_id=$1
  shift
  local pkgs=("$@")
    
  if [ "${#pkgs[@]}" -eq 0 ]; then
    return 0
  fi

  case "$os_id" in
    ubuntu|debian)
      sudo apt-get update && sudo apt-get install -y "${pkgs[@]}"
      ;;
    rhel|centos|fedora|almalinux|rocky)
      # Map package names for RHEL if necessary
      local rhel_pkgs=()
      for pkg in "${pkgs[@]}"; do
        if [[ "$pkg" == "python3-dev" ]]; then
          rhel_pkgs+=("python3-devel")
        else
          rhel_pkgs+=("$pkg")
        fi
      done
      sudo yum install -y "${rhel_pkgs[@]}"
      ;;
    arch|manjaro)
      # Map package names for Arch
      local arch_pkgs=()
      for pkg in "${pkgs[@]}"; do
        case "$pkg" in
          python3|python3-dev) arch_pkgs+=("python") ;;
          python3-setuptools) arch_pkgs+=("python-setuptools") ;;
          *) arch_pkgs+=("$pkg") ;;
        esac
      done
      sudo pacman -Sy --noconfirm && sudo pacman -S --noconfirm "${arch_pkgs[@]}"
      ;;
    *)
      echo "Error: Unsupported OS ID for package installation: $os_id"
      return 1
      ;;
  esac
}
