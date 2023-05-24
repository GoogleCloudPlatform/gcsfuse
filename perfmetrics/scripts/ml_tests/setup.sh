#!/bin/bash -i

# To run the bash Script.
# >> source setup.sh

# Go version to be installed.
GO_VERSION=go1.20.4.linux-amd64.tar.gz

# This function will install the given module/dependency if it's not alredy 
# installed.
function install {
  if [ "$(which $1)" = "" ] ;then
    sudo apt-get update -y
    echo "Installing $1..."
    if sudo apt-get install $*; then
      echo "Sucessfully installed $1"
    else
      echo "Failed to install $1. Please try again"
      exit 1
    fi
  fi
}

echo "Installing ops-agent..."
curl -sSO https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh
if sudo bash add-google-cloud-ops-agent-repo.sh --also-install; then
  echo "Sucessfully installed ops-agent"
else 
  echo "Failed to install ops-agent"
  exit 1
fi

# Install Golang.
if [ "$(which go)" = "" ] ;then
  echo "Installing Golang..."
  sudo apt-get update
  if sudo apt install wget -y; then
    wget https://golang.org/dl/$GO_VERSION
    sudo tar -zxvf $GO_VERSION
    if [ "$(which go)" = "" ]; then
      echo 'export GOROOT=$HOME/go' >> ~/.bashrc
      echo 'export GOPATH=$HOME/go' >> ~/.bashrc
      echo 'export PATH=$PATH:$GOROOT/bin:$GOPATH/bin' >> ~/.bashrc 
      source ~/.bashrc 
      echo "Sucessfully installed Golang"
    else
      echo "Failed to install Golang. Please try again"
      exit 1
    fi
  else
    echo "Failed to install wget. Please try again"
    exit 1
  fi
fi

# Install fuse.
install fuse -y

#Install git.
install git -y

#Install python3.9.
install python3.9 -y

echo "Install/Upgrade Prerequistes for automation script.."
sudo apt-get update
if sudo apt install python3-pip python-dev -y &&  
   sudo -H pip3 install --upgrade pip && pip3 install absl-py; then
  echo "You are now ready to run automation script"
else
  echo "Failed to install required Prerequistes for automation script. Please try again"
  exit 1
fi
