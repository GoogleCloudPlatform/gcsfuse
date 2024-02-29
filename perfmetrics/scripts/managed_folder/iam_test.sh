#!/bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License" 2>&1);
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# cd <gcsfuse dir>
# bash perfmetrics/scripts/managed_folder/iam_test.sh  SERVICE_ACCOUNT BUCKET_NAME MNT_DIR

SERVICE_ACCOUNT=$1
BUCKET_NAME=$2
MNT_DIR=$3

echo "Clean up permissions if already have..."
gcloud storage buckets remove-iam-policy-binding  gs://$BUCKET_NAME --member=serviceAccount:$SERVICE_ACCOUNT --role=roles/storage.objectViewer
gcloud storage buckets remove-iam-policy-binding  gs://$BUCKET_NAME --member=serviceAccount:$SERVICE_ACCOUNT --role=roles/storage.objectAdmin
gcloud alpha storage managed-folders rm -r gs://$BUCKET_NAME/managed_folder
gcloud alpha storage managed-folders create gs://$BUCKET_NAME/managed_folder

function permission_denied() {
  if [[ $result != *"Permission denied"* ]]; then
    echo "Command should return permission denied..."
    exit 1
  fi
}

function no_permission_denied() {
  if [[ $result == *"Permission denied"* ]]; then
    echo "Command should not return permission denied..."
    exit 1
  fi
}

function attach_permissions_to_managed_folder() {
  gcloud iam service-accounts keys create ~/managed_folder_key.json --iam-account=$SERVICE_ACCOUNT
  echo '{
    "bindings":[
      {
        "role": '"$PERMISSION"',
        "members":[
          "serviceAccount:'"$SERVICE_ACCOUNT"'"
        ]
      }
    ]
  }'> managed_folder_role.json
}

set -e
echo "1st Experiment, Bucket has no permission, managed folder has storage.objectViewer permission..."
PERMISSION="roles/storage.objectViewer"
attach_permissions_to_managed_folder
gcloud alpha storage managed-folders set-iam-policy gs://$BUCKET_NAME/managed_folder managed_folder_role.json
rm managed_folder_role.json
sudo umount $MNT_DIR
result=$(go run . --implicit-dirs -key-file ~/managed_folder_key.json  $BUCKET_NAME $MNT_DIR)
if [[ $result != *"exit status 1"* ]]; then
    echo "Bucket mounting should fail due to no permissions."
    exit 1
fi
go run .  --implicit-dirs --key-file ~/managed_folder_key.json  --only-dir managed_folder $BUCKET_NAME $MNT_DIR
echo "Managed folder will mount with view permission..."
result=$(ls $MNT_DIR 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/test.txt 2>&1)
permission_denied
sleep 15
sudo umount $MNT_DIR
go run .  --implicit-dirs --key-file ~/managed_folder_key.json  $MNT_DIR
echo "Dynamic mounting on managed folder should fail..."
result=$(ls $MNT_DIR/$BUCKET_NAME/managed_folder 2>&1)
permission_denied
sleep 15
sudo umount $MNT_DIR

echo "2nd Experiment, Bucket has storage.objectViewer permission, managed folder has no permission..."
gcloud alpha storage managed-folders remove-iam-policy-binding  gs://$BUCKET_NAME/managed_folder --member=serviceAccount:$SERVICE_ACCOUNT --role=roles/storage.objectViewer
gcloud storage buckets add-iam-policy-binding gs://$BUCKET_NAME --member=serviceAccount:$SERVICE_ACCOUNT --role=roles/storage.objectViewer
go run . --implicit-dirs -key-file ~/managed_folder_key.json  $BUCKET_NAME $MNT_DIR
echo "Bucket is mounted with viewer permission..."
result=$(ls $MNT_DIR 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/test.txt 2>&1)
sleep 15
sudo umount $MNT_DIR
go run . --implicit-dirs --key-file ~/managed_folder_key.json  --only-dir managed_folder $BUCKET_NAME $MNT_DIR
echo "Managed folder is mounted with viewer permission..."
result=$(ls $MNT_DIR 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/test.txt 2>&1)
sleep 15
sudo umount $MNT_DIR
go run .  --implicit-dirs --key-file ~/managed_folder_key.json  $MNT_DIR
echo "Dynamic mounting of managed folder is working with viewer permission..."
result=$(ls $MNT_DIR/$BUCKET_NAME/managed_folder 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/$BUCKET_NAME/test.txt 2>&1)
permission_denied
result=$(ls $MNT_DIR/$BUCKET_NAME 2>&1)
result=$(touch $MNT_DIR/$BUCKET_NAME/managed_folder/test2.txt 2>&1)
permission_denied
sleep 15
sudo umount $MNT_DIR

echo "3rd experiment, Bucket has storage.objectViewer permission, managed folder has storage.objectViewer permission..."
PERMISSION="roles/storage.objectViewer"
attach_permissions_to_managed_folder()
gcloud alpha storage managed-folders set-iam-policy gs://$BUCKET_NAME/managed_folder managed_folder_role.json
rm managed_folder_role.json
go run . --implicit-dirs -key-file ~/managed_folder_key.json  $BUCKET_NAME $MNT_DIR
echo "Bucket is mounted with viewer permissions...."
result=$(ls $MNT_DIR 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/test.txt 2>&1)
permission_denied
sleep 15
sudo umount $MNT_DIR
go run .  --implicit-dirs --key-file ~/managed_folder_key.json  --only-dir managed_folder $BUCKET_NAME $MNT_DIR
echo "Managed folder is mounted with viewer permissions..."
result=$(ls $MNT_DIR 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/test.txt 2>&1)
permission_denied
sleep 15
sudo umount $MNT_DIR
go run .  --implicit-dirs --key-file ~/managed_folder_key.json  $MNT_DIR
echo "Dynamic mounting of managed folder is working with viewer permission..."
result=$(ls $MNT_DIR/$BUCKET_NAME/managed_folder 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/$BUCKET_NAME/test.txt 2>&1)
permission_denied
result=$(ls $MNT_DIR/$BUCKET_NAME 2>&1)
result=$(touch $MNT_DIR/$BUCKET_NAME/managed_folder/test.txt 2>&1)
permission_denied
sleep 15
sudo umount $MNT_DIR

echo "4th experiment, Bucket has storage.objectViewer permission, managed folder has storage.objectAdmin permission"
gcloud alpha storage managed-folders remove-iam-policy-binding  gs://$BUCKET_NAME/managed_folder --member=serviceAccount:$SERVICE_ACCOUNT --role=roles/storage.objectViewer
PERMISSION="roles/storage.objectAdmin"
attach_permissions_to_managed_folder()
gcloud alpha storage managed-folders set-iam-policy gs://$BUCKET_NAME/managed_folder managed_folder_role.json
rm managed_folder_role.json
go run . --implicit-dirs -key-file ~/managed_folder_key.json  $BUCKET_NAME $MNT_DIR
echo "Bucket will mount with view permission"
result=$(ls $MNT_DIR 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/test.txt 2>&1)
permission_denied
sleep 15
sudo umount $MNT_DIR
go run .  --implicit-dirs --key-file ~/managed_folder_key.json  --only-dir managed_folder $BUCKET_NAME $MNT_DIR
echo "Managed folder is mounted with admin permissions..."
result=$(ls $MNT_DIR 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/test.txt 2>&1)
no_permission_denied
sleep 15
sudo umount $MNT_DIR
go run .  --implicit-dirs --key-file ~/managed_folder_key.json  $MNT_DIR
result=$(ls $MNT_DIR/$BUCKET_NAME/managed_folder 2>&1)
no_permission_denied
echo "Bucket has only view permissions"
result=$(touch $MNT_DIR/$BUCKET_NAME/test.txt 2>&1)
permission_denied
result=$(ls $MNT_DIR/$BUCKET_NAME/managed_folder 2>&1)
echo "Dynamic mounting of managed folder is working with admin permission..."
result=$(touch $MNT_DIR/$BUCKET_NAME/managed_folder/test.txt 2>&1)
no_permission_denied
result=$(rm $MNT_DIR/$BUCKET_NAME/managed_folder/test.txt 2>&1)
no_permission_denied
sleep 15
sudo umount $MNT_DIR

echo "5th experiment, Bucket has storage.objectAdmin permission, managed folder has storage.objectAdmin permission"
gcloud storage buckets remove-iam-policy-binding  gs://$BUCKET_NAME --member=serviceAccount:$SERVICE_ACCOUNT --role=roles/storage.objectViewer
gcloud storage buckets add-iam-policy-binding gs://$BUCKET_NAME --member=serviceAccount:$SERVICE_ACCOUNT --role=roles/storage.objectAdmin
go run . --implicit-dirs -key-file ~/managed_folder_key.json  $BUCKET_NAME $MNT_DIR
echo "Bucket is mounted with admin permission..."
result=$(ls $MNT_DIR 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/test.txt 2>&1)
no_permission_denied
result=$(rm $MNT_DIR/test.txt 2>&1)
no_permission_denied
sleep 15
sudo umount $MNT_DIR
go run .  --implicit-dirs --key-file ~/managed_folder_key.json  --only-dir managed_folder $BUCKET_NAME $MNT_DIR
echo "Managed folder is mounted with admin permissions..."
result=$(ls $MNT_DIR 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/test.txt 2>&1)
no_permission_denied
sleep 15
sudo umount $MNT_DIR
go run .  --implicit-dirs --key-file ~/managed_folder_key.json  $MNT_DIR
result=$(ls $MNT_DIR/$BUCKET_NAME/managed_folder 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/$BUCKET_NAME/test.txt 2>&1)
no_permission_denied
result=$(rm $MNT_DIR/$BUCKET_NAME/test.txt 2>&1)
no_permission_denied
echo "Dynamic mounting of managed folder is working with admin permission..."
result=$(touch $MNT_DIR/$BUCKET_NAME/managed_folder/test.txt 2>&1)
no_permission_denied
result=$(rm $MNT_DIR/$BUCKET_NAME/managed_folder/test.txt 2>&1)
no_permission_denied
sleep 15
sudo umount $MNT_DIR

echo "6th experiment, Bucket has storage.objectAdmin permission, managed folder has storage.objectViewer permission"
gcloud alpha storage managed-folders remove-iam-policy-binding  gs://$BUCKET_NAME/managed_folder --member=serviceAccount:$SERVICE_ACCOUNT --role=roles/storage.objectAdmin
PERMISSION="roles/storage.objectViewer"
attach_permissions_to_managed_folder()
gcloud alpha storage managed-folders set-iam-policy gs://$BUCKET_NAME/managed_folder managed_folder_role.json
rm managed_folder_role.json
go run . --implicit-dirs -key-file ~/managed_folder_key.json  $BUCKET_NAME $MNT_DIR
echo "Bucket is mounted with admin permission..."
result=$(ls $MNT_DIR 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/test.txt 2>&1)
no_permission_denied
result=$(rm $MNT_DIR/test.txt 2>&1)
no_permission_denied
sleep 15
sudo umount $MNT_DIR
go run .  --implicit-dirs --key-file ~/managed_folder_key.json  --only-dir managed_folder $BUCKET_NAME $MNT_DIR
echo "Managed folder is mounted with admin permissions..."
result=$(ls $MNT_DIR 2>&1)
no_permission_denied
result=$(touch $MNT_DIR/test.txt 2>&1)
no_permission_denied
sleep 15
sudo umount $MNT_DIR
go run .  --implicit-dirs --key-file ~/managed_folder_key.json  $MNT_DIR
result=$(ls $MNT_DIR/$BUCKET_NAME/managed_folder 2>&1)
no_permission_denied
echo "Bucket has only admin permissions"
touch $MNT_DIR/$BUCKET_NAME/test.txt
result=$(rm $MNT_DIR/$BUCKET_NAME/test.txt 2>&1)
no_permission_denied
echo "Dynamic mounting of managed folder is working with admin permission..."
result=$(touch $MNT_DIR/$BUCKET_NAME/managed_folder/test.txt 2>&1)
no_permission_denied
result=$(rm $MNT_DIR/$BUCKET_NAME/managed_folder/test.txt 2>&1)
no_permission_denied
sleep 15
sudo umount $MNT_DIR
