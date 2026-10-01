# Step one: gcloud command to create cache on bucket name
gcloud alpha storage buckets anywhere-caches create gs://<bucket-name> <same-zone-as-VM>

# Step two: run fio scripts ./fio_scripts

# Step three: To validate if data is being consumed from cache, please use "anywhere-cache" section of VM observability on pantheon
# https://cloud.google.com/storage/nda-docs/using-anywhere-cache#monitor-performance

#Step four: Once the run is complete and validated, please disable the cache and remove the bucket using following command
gcloud storage buckets anywhere-caches disable <cache-id> # Cache is formulated from bucketname/zone
gcloud storage buckets delete <bucket_name>