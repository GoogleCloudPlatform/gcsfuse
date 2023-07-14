#! /bin/bash
set -x
gsutil cp  gs://gcsfuse-release-packages/version-detail/details.txt .
curl http://metadata.google.internal/computeMetadata/v1/instance/name -H "Metadata-Flavor: Google" >> details.txt

if grep -q ubuntu details.txt || grep -q debian details.txt;
then
    sudo adduser --ingroup google-sudoers --disabled-password --home=/home/starterscript --gecos "" starterscript
else
    sudo adduser -g google-sudoers --home-dir=/home/starterscript starterscript
fi
sudo -u starterscript bash -c '
set -x
cd ~/

gsutil cp  gs://gcsfuse-release-packages/version-detail/details.txt .
curl http://metadata.google.internal/computeMetadata/v1/instance/name -H "Metadata-Flavor: Google" >> details.txt
touch ~/logs.txt
echo User: $USER &>> ~/logs.txt
echo Current Working Directory: $(pwd)  &>> ~/logs.txt

if grep -q ubuntu details.txt || grep -q debian details.txt;
then
    sudo apt update

    #Install fuse
    yes | sudo apt install fuse

    # download and install gcsfuse deb package
    gsutil cp gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/gcsfuse_$(sed -n 1p details.txt)_amd64.deb .
    yes | sudo dpkg -i gcsfuse_$(sed -n 1p details.txt)_amd64.deb |& tee -a ~/logs.txt

    # install wget
    yes | sudo apt install wget

    #install git
    yes | sudo apt install git

    #install build-essentials
    yes | sudo apt install build-essential
else
    sudo yum makecache
    sudo yum check-update

    #Install fuse
    yes | sudo yum install fuse

    #download and install gcsfuse rpm package
    gsutil cp gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/gcsfuse-$(sed -n 1p details.txt)-1.x86_64.rpm .
    yes | sudo yum localinstall gcsfuse-$(sed -n 1p details.txt)-1.x86_64.rpm

    #install wget
    yes | sudo yum install wget

    #install git
    yes | sudo yum install git

    #install Development tools
    yes | sudo yum install gcc gcc-c++ make
fi

# install go
wget -O go_tar.tar.gz https://go.dev/dl/go1.20.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go_tar.tar.gz
export PATH=${PATH}:/usr/local/go/bin

#Write gcsfuse and go version to log file
gcsfuse --version |& tee -a ~/logs.txt
go version |& tee -a ~/logs.txt

# Clone and checkout gcsfuse repo
export PATH=${PATH}:/usr/local/go/bin
git clone https://github.com/googlecloudplatform/gcsfuse |& tee -a ~/logs.txt
cd gcsfuse
git checkout $(sed -n 2p ~/details.txt) |& tee -a ~/logs.txt

#run tests with testbucket flag
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/... -p 1 --integrationTest -v --testbucket=$(sed -n 3p ~/details.txt) --testInstalledPackage --timeout=60m &>> ~/logs.txt

if grep -q FAIL ~/logs.txt; then
    echo "Test failures detected" &>> ~/logs.txt
else
    touch success.txt
    gsutil cp success.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
fi

gsutil cp ~/logs.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
'
