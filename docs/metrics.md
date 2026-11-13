# GCSFuse Metrics

For instructions on how to enable and use Cloud Storage FUSE metrics, refer to the metrics guide at <https://cloud.google.com/storage/docs/cloud-storage-fuse/metrics>.

## Experimental OpenTelemetry endpoints

The `metrics.experimental-otel-metrics-endpoint` and
`logging.experimental-otel-logging-endpoint` settings accept a hostname or IP
address with an optional port, not a URL, scheme, or path. IPv6 addresses must be
bracketed, for example `[::1]:4318`.

Google authentication is selected only for `googleapis.com` and its subdomains.
DNS names use ASCII case-insensitive comparison, ignoring one trailing dot. Plaintext
HTTP is selected automatically only for `localhost`, literal loopback IP addresses
(including `127.0.0.0/8` and `::1`), and the IPv4 wildcard address `0.0.0.0`, retained
for local collector testing. Hostnames are not resolved to decide these rules;
names such as `localhost.collector.example` and
`googleapis.com.collector.example` do not qualify.
