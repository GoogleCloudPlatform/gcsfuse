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

        sudo apt-get update && sudo apt-get install --only-upgrade gcsfuse

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

## Install gcsfuse by building the latest source code:
Make sure you have `fuse`, `git` and `go` (the newest version specified in [go.mod](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/go.mod)) installed on the system.

### Method 1: Using the go install command directly
Install gcsfuse directly using the below command:
```
go install github.com/googlecloudplatform/gcsfuse@master
```

### Method 2: By cloning the git repo
1. Clone the repo using:
```
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
```
2. Change to repo directory:
```
cd gcsfuse
```
3. Install gcsfuse:
```
go install .
```

**Note:** In both cases, a binary named `gcsfuse` will be installed to `$GOPATH/bin`.