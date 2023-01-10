#!/bin/bash
# TODO: Enable the logs once log rotation feature is merged
gcsfuse --implicit-dirs --max-conns-per-host 100 --disable-http2 --log-format text --foreground --stackdriver-export-interval 60s $BUCKET_NAME myBucket > /home/output/gcsfuse.out 2> /home/output/gcsfuse.err &
