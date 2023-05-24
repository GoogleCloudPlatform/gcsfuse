gcloud config set project gcs-fuse-test
gcloud compute instances delete --quiet tf-resnet-7d --zone us-central1-c
gcloud compute instances create tf-resnet-7d \
    --project=gcs-fuse-test \
    --zone=us-central1-c \
    --machine-type=a2-highgpu-2g \
    --network-interface=network-tier=PREMIUM,nic-type=GVNIC,stack-type=IPV4_ONLY,subnet=default \
    --metadata=enable-oslogin=true \
    --maintenance-policy=TERMINATE \
    --provisioning-model=STANDARD \
    --service-account=927584127901-compute@developer.gserviceaccount.com \
    --scopes=https://www.googleapis.com/auth/cloud-platform \
    --accelerator=count=2,type=nvidia-tesla-a100 \
    --create-disk=auto-delete=yes,boot=yes,device-name=tf-resnet-7d,image=projects/ubuntu-os-cloud/global/images/ubuntu-2004-focal-v20230523,mode=rw,size=200,type=projects/gcs-fuse-test/zones/us-central1-c/diskTypes/pd-balanced \
    --no-shielded-secure-boot \
    --shielded-vtpm \
    --shielded-integrity-monitoring \
    --labels=goog-ec-src=vm_add-gcloud \
    --reservation-affinity=any
sleep 60s

gcloud compute ssh tf-resnet-7d --zone us-central1-c --command "mkdir github; cd github; git clone https://github.com/GoogleCloudPlatform/gcsfuse.git; cd gcsfuse; git checkout ai_ml_tests;" --internal-ip
gcloud compute ssh tf-resnet-7d --zone us-central1-c --command "cd github/gcsfuse/perfmetrics/scripts/continuous_test/ml_tests/tf/resnet/; export KOKORO_ARTIFACTS_DIR=\$HOME; bash build.sh 1> ~/a.out 2> ~/e.err &" --internal-ip
sleep 100s
gcloud compute ssh tf-resnet-7d --zone us-central1-c --command "cat a.out; cat e.err;" --internal-ip
