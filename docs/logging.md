# GCSFuse User Logging Guide

GCSFuse provides several options for logging, which can be configured via command-line flags or a configuration file. For a complete list of logging configurations, refer to the GCSFuse configuration file documentation.

## Log Destinations

You can direct GCSFuse logs to one of the following destinations:

*   **Standard Output (stdout):** When running GCSFuse in the foreground (with the `--foreground` flag) and without a log file specified, logs are printed to stdout.
*   **Syslog:** When running GCSFuse in the background (default) and without a log file specified, logs are sent to syslog.
*   **File:** You can specify a log file path using the `--log-file` flag or `logging:file-path` config option. This is useful for persistent logging and for integration with log collectors like `fluentd`.
*   **Google Cloud Logging:** (Experimental) GCSFuse can send logs directly to Google Cloud Logging. This is useful for centralized logging and analysis within Google Cloud.

### Enabling Google Cloud Logging

To enable logging directly to Google Cloud Logging, use the `--experimental-enable-cloud-logging` flag or set `logging:experimental-enable-cloud-logging: true` in your configuration file.

**Requirements:**
*   The environment where GCSFuse is running must have credentials to write to Cloud Logging. This is typically handled automatically on Google Cloud environments like GCE or GKE by assigning the `roles/logging.logWriter` IAM role to the service account.
*   The log format must be `json`. GCSFuse will use this format by default when Cloud Logging is enabled.

When enabled, this option is mutually exclusive with file-based logging (`--log-file`).