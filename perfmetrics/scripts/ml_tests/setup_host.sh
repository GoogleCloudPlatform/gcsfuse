#!/bin/bash
# This file installs docker engine and nvidia driver and nvidia container tool
# necessary for running dlc container on the vm

# Install Ops-agent to get the memory and processes' related data on VM console.
curl -sSO https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh

# Add the pub_key, for the package verification while installing using apt-get.
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys B53DC80D13EDEF05
sudo bash add-google-cloud-ops-agent-repo.sh --also-install --version=latest

# Steps to install nvidia-driver, docker-engine, nvidia-docker2 and their required dependencies
echo "Installing linux utility packages, like, lsb-release, curl..."
sudo apt-get update
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

echo "Installing docker framework..."
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin -y

echo "Installing driver..."
sudo apt update && sudo apt install -y build-essential
BASE_URL=https://us.download.nvidia.com/tesla
DRIVER_VERSION=450.172.01
sudo curl -fSsl -O $BASE_URL/$DRIVER_VERSION/NVIDIA-Linux-x86_64-$DRIVER_VERSION.run

sudo sh NVIDIA-Linux-x86_64-$DRIVER_VERSION.run -s

echo "Installing NVIDIA container tool..."
distribution=$(. /etc/os-release;echo $ID$VERSION_ID) \
      && curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
      && curl -s -L https://nvidia.github.io/libnvidia-container/$distribution/libnvidia-container.list | \
            sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
            sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list

sudo apt-get update
sudo apt-get install -y nvidia-docker2
sudo systemctl restart docker
