# Basic usage

First, ensure that you have downloaded a JSON file containing a private key for
your [service account][] from the [Google Developers Console][console]. Say for the purposes of this document that it is located at `/path/to/key.json`.

[service account]: https://cloud.google.com/storage/docs/authentication#service_accounts
[console]: https://console.developers.google.com

Next, create the directory into which you want to mount the gcsfuse bucket:

    mkdir /path/to/mount/point
