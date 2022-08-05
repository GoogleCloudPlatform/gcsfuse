#!/bin/bash
set -e
echo Installing pip and fuse..
sudo apt-get install fuse -y
sudo apt-get install pip -y
echo Installing requirements..
pip install -r requirements.txt --user
echo Running script..
python3 listing_benchmark.py config.json --command "ls -R" --num_samples 30 --upload --message "Testing CT setup."
