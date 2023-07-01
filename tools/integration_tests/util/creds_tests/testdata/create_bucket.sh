gcloud storage buckets create gs://$1 --project "gcs-fuse-test" --location="us-west1"
gsutil iam ch serviceAccount:$2:objectCreator gs://$1