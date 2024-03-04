gcloud version
which gcloud
sudo rm -rf $(which gcloud)
sudo rm /snap/bin/gcloud
curl -o gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz
sudo tar xzf gcloud.tar.gz
sudo ./google-cloud-sdk/install.sh
PATH="$PATH:google-cloud-sdk/bin"
echo $PATH
which gcloud
gcloud components update
gcloud components install alpha
gcloud version
sudo google-cloud-sdk/bin/gcloud alpha storage managed-folders create gs://new-testing-tulsishah/m_test
sleep 10000