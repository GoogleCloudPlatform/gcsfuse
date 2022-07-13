#i!/bin/bash
set -e
sudo apt-get update

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

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"

echo "Setting up a machine"
chmod +x ml_tests/setup.sh
source ml_tests/setup.sh