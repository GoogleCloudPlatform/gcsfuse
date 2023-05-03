Install Cloud Storage FUSE and its dependencies using prebuilt Linux binaries.

If you are running Linux on a 64-bit x86 machine and want to install pre-built binaries (i.e. you don't want to build from source), you must have [FUSE](https://github.com/libfuse/libfuse)  installed. You can then download and install the latest release package. The instructions vary by distribution

## Install on Ubuntu or Debian

To install Cloud Storage FUSE for Ubuntu or Debian, follow the instructions below:

1.  Add the Cloud Storage FUSE distribution URL as a package source and import its public key, or download the package directly from [GitHub](https://github.com/GoogleCloudPlatform/gcsfuse/releases):


        export GCSFUSE_REPO=gcsfuse-`lsb_release -c -s`
        echo "deb https://packages.cloud.google.com/apt $GCSFUSE_REPO main" | sudo tee /etc/apt/sources.list.d/gcsfuse.list
        curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -

2.  Update the list of packages available and install Cloud Storage FUSE:

        sudo apt-get update
        sudo apt-get install gcsfuse

3. Cloud Storage FUSE updates can be installed by:

        sudo apt-get update && sudo apt-get install –-only-upgrade gcsfuse

## Install on CentOS or Red Hat

To install Cloud Storage FUSE for CentOS or Red Hat, follow the instructions below.

1. Configure the Cloud Storage FUSE repository and its associated public key:

       sudo tee /etc/yum.repos.d/gcsfuse.repo > /dev/null <<EOF
       [gcsfuse]
       name=gcsfuse (packages.cloud.google.com)
       baseurl=https://packages.cloud.google.com/yum/repos/gcsfuse-el7-x86_64
       enabled=1
       gpgcheck=1
       repo_gpgcheck=0
       gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
              https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
       EOF

2. Install Cloud Storage FUSE using the YUM Linux package manager:

       sudo yum install gcsfuse

   Be sure to answer "yes" to any questions about adding the GPG signing key.

# Install by building the binaries from source

To install Cloud Storage FUSE by building the binaries from source, follow the instructions below:

1. Make sure you have a working Go installation, the newest version specified in [go.mod](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/go.mod). See Installing Go from source.
2. [Fuse](https://github.com/libfuse/libfuse). See the instructions for the binary release above.
3. Make sure you have the Git command-line tool installed. This is probably available as ```git``` in your package manager.
4. To install or update Cloud Storage FUSE, run the following command

       GO111MODULE=auto go get -u github.com/googlecloudplatform/gcsfuse

This will fetch the latest Cloud Storage FUSE sources to ```$GOPATH/src/github.com/googlecloudplatform/gcsfuse```, build the sources, and then install a binary named gcsfuse to ```$GOPATH/bin```.
