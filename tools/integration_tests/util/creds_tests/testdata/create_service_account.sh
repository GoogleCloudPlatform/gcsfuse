SERVICE_ACCOUNT=$1
gcloud iam service-accounts create $SERVICE_ACCOUNT --description="$SERVICE_ACCOUNT" --display-name="$SERVICE_ACCOUNT"