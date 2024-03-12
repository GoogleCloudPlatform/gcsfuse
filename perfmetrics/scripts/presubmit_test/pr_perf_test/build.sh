set -e
sleep 100000
gcloud version
wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
sudo rm -rf $(which gcloud) && sudo tar xzf gcloud.tar.gz && sudo mv google-cloud-sdk /usr/local
if [ ! -d $(which gcloud) ]; then
   sudo rm -rf $(which gcloud)
   echo "Hello" 
fi
sudo /usr/local/google-cloud-sdk/install.sh
export PATH=$PATH:/usr/local/google-cloud-sdk/bin
echo 'export PATH=$PATH:/usr/local/google-cloud-sdk/bin' >> ~/.bashrc
gcloud version && rm gcloud.tar.gz
sudo /usr/local/google-cloud-sdk/bin/gcloud components update
sudo /usr/local/google-cloud-sdk/bin/gcloud components install alpha
which gcloud
gcloud version
touch a.txt
gcloud alpha storage managed-folders create gs://write-test-gcsfuse-tulsishah/m1
sleep 100000
