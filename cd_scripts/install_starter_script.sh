#! /bin/bash
set -x
gsutil cp  gs://gcsfuse-release-packages/version-detail/details.txt .
curl http://metadata.google.internal/computeMetadata/v1/instance/name -H "Metadata-Flavor: Google" >> details.txt
touch ~/logs.txt

# Install apt-transport-artifact-registry
if grep -q ubuntu details.txt || grep -q debian details.txt;
then
    curl https://us-central1-apt.pkg.dev/doc/repo-signing-key.gpg | sudo apt-key add - && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
    echo 'deb http://packages.cloud.google.com/apt apt-transport-artifact-registry-stable main' | sudo tee -a /etc/apt/sources.list.d/artifact-registry.list
    sudo apt update
    sudo apt install apt-transport-artifact-registry
    echo 'deb ar+https://us-apt.pkg.dev/projects/gcs-fuse-prod gcsfuse-bullseye main' | sudo tee -a /etc/apt/sources.list.d/artifact-registry.list
    sudo apt update

    # Install released gcsfuse version
    yes | sudo apt install gcsfuse=$(sed -n 1p details.txt) -t gcsfuse-bullseye |& tee -a ~/logs.txt
else
    sudo yum makecache
    yes | sudo yum install yum-plugin-artifact-registry
sudo tee -a /etc/yum.repos.d/artifact-registry.repo << EOF
[gcsfuse-el7-x86-64]
name=gcsfuse-el7-x86-64
baseurl=https://asia-yum.pkg.dev/projects/gcs-fuse-prod/gcsfuse-el7-x86-64
enabled=1
repo_gpgcheck=0
gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
    https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF
    sudo yum makecache
    yes | sudo yum --enablerepo=gcsfuse-el7-x86-64 install gcsfuse-$(sed -n 1p details.txt)-1 |& tee -a ~/logs.txt
fi

# Verify gcsfuse version (successful installation)
gcsfuse --version |& tee version.txt
installed_version=$(echo $(sed -n 1p version.txt) | cut -d' ' -f3)
if grep -q $installed_version details.txt; then
    echo "GCSFuse latest version installed correctly." &>> ~/logs.txt
else
    echo "Failure detected in latest gcsfuse version installation." &>> ~/logs.txt
fi

# Uninstall gcsfuse latest version and install old version
if grep -q ubuntu details.txt || grep -q debian details.txt;
then
    yes | sudo apt remove gcsfuse
    yes | sudo apt install gcsfuse=0.42.5 -t gcsfuse-bullseye |& tee -a ~/logs.txt
else
    yes | sudo yum remove gcsfuse
    yes | sudo yum install gcsfuse-0.42.5-1 |& tee -a ~/logs.txt
fi

# verify old version installation
gcsfuse --version |& tee version.txt
installed_version=$(echo $(sed -n 1p version.txt) | cut -d' ' -f3)
if [ $installed_version == "0.42.5" ]; then
    echo "GCSFuse old version (0.42.5) installed successfully" &>> ~/logs.txt
else
    echo "Failure detected in GCSFuse old version installation." &>> ~/logs.txt
fi

# Upgrade gcsfuse to latest version
if grep -q ubuntu details.txt || grep -q debian details.txt;
then
    sudo apt install --only-upgrade gcsfuse |& tee -a ~/logs.txt
else
    yes | sudo yum upgrade gcsfuse |& tee -a ~/logs.txt
fi

gcsfuse --version |& tee version.txt
installed_version=$(echo $(sed -n 1p version.txt) | cut -d' ' -f3)
if grep -q $installed_version details.txt; then
    echo "GCSFuse successfully upgraded to latest version $installed_version." &>> ~/logs.txt
else
    echo "Failure detected in upgrading to latest gcsfuse version." &>> ~/logs.txt
fi

if grep -q Failure ~/logs.txt; then
    echo "Test failed" &>> ~/logs.txt ;
else
    touch success.txt
    gsutil cp success.txt gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/installation-test/$(sed -n 3p details.txt)/   ;
fi

gsutil cp ~/logs.txt gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/installation-test/$(sed -n 3p details.txt)/