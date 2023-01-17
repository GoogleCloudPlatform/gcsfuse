#!/bin/bash
# TODO: Enable the logs once log rotation feature is merged
go run . --implicit-dirs --max-conns-per-host 100 --disable-http2 --log-format text --stackdriver-export-interval 60s $BUCKET_NAME myBucket > /home/output/gcsfuse.out 2> /home/output/gcsfuse.err &
