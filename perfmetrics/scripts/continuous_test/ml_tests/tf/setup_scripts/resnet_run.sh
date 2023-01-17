#!/bin/bash
sudo docker exec tf_model_container bash -c 'BUCKET_NAME=ml-models-data-gcsfuse ./gcsfuse_mount.sh'
sudo docker exec tf_model_container sh -c 'nohup python3 -u resnet.py > /home/output/myprogram.out 2> /home/output/myprogram.err &'
