#!/bin/bash
sudo docker exec tf_model_container bash -c 'BUCKET_NAME=ml-models-data-gcsfuse /dlc_testing/gcsfuse_mount.sh'
sudo docker exec tf_model_container bash -c 'BUCKET_NAME=ml-models-data-gcsfuse /dlc_testing/gcsfuse_mount.sh'