#!/bin/bash


# gcloud iam service-accounts --list

gcloud iam service-accounts create ssiog-runner --display-name="ssiog-runner"

gcloud projects add-iam-policy-binding gcs-tess \
  --member="serviceAccount:ssiog-runner@gcs-tess.iam.gserviceaccount.com" \
  --role="roles/storage.objectUser"


gcloud projects add-iam-policy-binding gcs-tess \
  --member="serviceAccount:ssiog-runner@gcs-tess.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.reader"

gcloud projects add-iam-policy-binding gcs-tess \
  --member="serviceAccount:ssiog-runner@gcs-tess.iam.gserviceaccount.com" \
  --role="roles/logging.logWriter"

gcloud iam service-accounts add-iam-policy-binding ssiog-runner@gcs-tess.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:gcs-tess.svc.id.goog[default/princer-ssiog-ksa-e2892743606541ae]"

kubectl annotate serviceaccount princer-ssiog-ksa-e2892743606541ae \
  --namespace default \
  iam.gke.io/gcp-service-account=ssiog-runner@gcs-tess.iam.gserviceaccount.com

  kubectl annotate serviceaccount ssiog-runner-ksa \
  --namespace default \
  iam.gke.io/gcp-service-account=ssiog-runner@gcs-tess.iam.gserviceaccount.com


# gcloud storage buckets add-iam-policy-binding gs://princer-ssiog-metrics-bkt     --member "principal://iam.googleapis.com/projects/222564316065/locations/global/workloadIdentityPools/gcs-tess.svc.id.goog/subject/ns/default/sa/princer-ssiog-ksa-e2892743606541ae"     --role "roles/storage.objectAdmin"

# gcloud storage buckets add-iam-policy-binding gs://princer-ssiog-data-bkt     --member "principal://iam.googleapis.com/projects/222564316065/locations/global/workloadIdentityPools/gcs-tess.svc.id.goog/subject/ns/default/sa/princer-ssiog-ksa-e2892743606541ae"     --role "roles/storage.objectAdmin"

gcloud iam service-accounts describe ssiog-runner@gcs-tess.iam.gserviceaccount.com --project=gcs-tess --format=yaml

gcloud iam service-accounts get-iam-policy \
  ssiog-runner@gcs-tess.iam.gserviceaccount.com \
  --project=gcs-tess \
  --format="value(bindings)"

gcloud iam service-accounts get-iam-policy \
  ssiog-runner@gcs-tess.iam.gserviceaccount.com \
  --project=gcs-tess \
  --format=yaml