# Use Ubuntu 22.04 as the base image
FROM ubuntu:22.04

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Update and install system dependencies
# git, curl, wget, sudo: utilities
# build-essential: for compiling code (gcc, etc.)
# python3, python3-dev, python3-setuptools: required by test scripts
# python3-crcmod: required by gsutil for composite object integrity checks
# fuse: required for gcsfuse
RUN apt-get update && apt-get install -y \
    git \
    curl \
    wget \
    sudo \
    build-essential \
    python3 \
    python3-dev \
    python3-setuptools \
    python3-crcmod \
    fuse \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.24.11
ENV GO_VERSION=1.24.11
RUN wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz" && \
    rm "go${GO_VERSION}.linux-amd64.tar.gz"

# Add Go to PATH
ENV PATH="/usr/local/go/bin:${PATH}"

# Install gcloud SDK
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key --keyring /usr/share/keyrings/cloud.google.gpg add - && \
    apt-get update && apt-get install -y google-cloud-sdk google-cloud-sdk-alpha && \
    rm -rf /var/lib/apt/lists/*

# Pre-install go-junit-report (before copying source to optimize caching)
RUN go install github.com/jstemmer/go-junit-report/v2@latest
ENV PATH="/root/go/bin:${PATH}"

# Set up workspace
WORKDIR /workspace

# Copy the repository
COPY . .

# Patch the internal_run_e2e_tests.sh to disable the CALL to install_packages in main()
# We found that `install_packages` is called at line 669 inside `main()`.
# We will use sed to comment out the line that matches exactly "  install_packages" (with two spaces indentation)
# or just "install_packages" if it's the only content of the line.
# A safe way is to replace the line containing the function call within the main block.
# Since we know the file content, we can target the call in the main function.
# The call is inside main(), preceded by `set -e`.
RUN sed -i 's/^\s*install_packages$/# install_packages/g' tools/integration_tests/internal_run_e2e_tests.sh

# Entrypoint
ENTRYPOINT ["tools/integration_tests/internal_run_e2e_tests.sh"]
