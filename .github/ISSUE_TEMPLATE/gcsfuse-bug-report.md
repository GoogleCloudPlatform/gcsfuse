---
name: GCSFuse Issue report
about: Describe information users can provide to enable faster interaction
title: ''
labels: "question,p2"
assignees: ''

---

**Describe the issue**
Please provide a clear description of what you were trying to achieve along with the details of the flags that you passed.

**System & Version (please complete the following information):**
 - OS: [e.g. Ubuntu 20.04]
 - Platform [GCE VM, GKE, Vertex AI]
 - Version [Gcsfuse version and GKE version]

**Steps to reproduce the behavior with following information:**
1. Please share Mount command including all command line or config flags used to mount the bucket.
2. Please make sure you have no other security, monitoring, background processes which can offend the FUSE process running. Possibly reproduce under a fresh/clean installation.
3. Please rerun with --log-severity=TRACE --foreground as additional flags to enable debug logs.
4. Monitor the logs and please capture screenshots or copy the relevant logs to a file (can use --log-format and --log-file as well).
5. Attach the screenshot or the logs file to the bug report here.
6. If you're using gcsfuse with any other library/tool/process please list out the steps you took to reproduce the issue.

**Additional context**
Add any other context about the problem here.

**SLO:**
We strive to respond to all bug reports within 24 business hours provided the information mentioned above is included.
