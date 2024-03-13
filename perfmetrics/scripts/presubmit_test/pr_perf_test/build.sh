set -e
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse
git checkout fix_e2e_tests_arm64_machine
bash perfmetrics/scripts/run_e2e_tests.sh false
