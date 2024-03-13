set -e
gcloud version
wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
sudo tar xzf gcloud.tar.gz && sudo mv google-cloud-sdk /usr/local
sudo /usr/local/google-cloud-sdk/install.sh
export PATH=/usr/local/google-cloud-sdk/bin:$PATH
echo 'export PATH=/usr/local/google-cloud-sdk/bin:$PATH' >> ~/.bashrc
gcloud version && rm gcloud.tar.gz
sudo /usr/local/google-cloud-sdk/bin/gcloud components update
sudo /usr/local/google-cloud-sdk/bin/gcloud components install alpha
echo "gcloud version"
gcloud version
