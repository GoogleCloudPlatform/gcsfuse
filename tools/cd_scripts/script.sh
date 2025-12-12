gcloud storage cp details.txt gs://ashmeenbkt/version-detail/details.txt

vm_name=ashmeen-release-test-rhel-9-arm64-1

#gcloud storage buckets create gs://${vm_name} --project=gcs-fuse-test --location=us-central1
#gcloud storage buckets create gs://${vm_name}-hns --project=gcs-fuse-test --location=us-central1 --uniform-bucket-level-access --enable-hierarchical-namespace
#gcloud storage buckets create gs://${vm_name}-parallel --project=gcs-fuse-test --location=us-central1
#gcloud storage buckets create gs://${vm_name}-hns-parallel --project=gcs-fuse-test --location=us-central1 --uniform-bucket-level-access --enable-hierarchical-namespace

gcloud compute instances delete ${vm_name} --zone=us-central1-f
gcloud compute instances create ${vm_name} \
    --machine-type=t2a-standard-48 \
    --image-project=rhel-cloud --zone=us-central1-f \
    --scopes=https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/devstorage.read_write \
    --boot-disk-size=75GiB \
    --reservation-affinity=specific \
    --metadata run-on-zb-only=false,run-read-cache-only=false,run-light-test=true \
    --metadata-from-file=startup-script=e2e_test.sh \
    --image=rhel-9-arm64-v20250812 \
    --reservation=projects/gcs-fuse-test/reservations/release-e2e-test-ubuntu-2310-arm64
#    --reservation=projects/gcs-fuse-test/reservations/release-e2e-test-debian-11-arm64