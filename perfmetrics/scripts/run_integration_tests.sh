# Mounting gcsfuse for running integration tests for the mountedDirectory flag
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100"
BUCKET_NAME=gcsfuse-integration-tests
MOUNT_POINT=gcs
mkdir -p gcs
# The VM will itself exit if the gcsfuse mount fails.
gcsfuse $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
# Executing integration tests for the mountedDirectory flag
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/... --integrationTest -v --mountedDirectory=../../../$MOUNT_POINT
sudo umount gcs
