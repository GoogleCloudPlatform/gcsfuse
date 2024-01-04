---
name: GCSFuse Issue report
about: Describe information users can provide to enable faster interaction
title: ''
labels: "question,p1"
assignees: ''

---

**Describe the issue**
Please provide a clear description of what you were trying to achieve along with the details of the flags that you passed.

**To Collect more Debug logs**
Steps to reproduce the behavior:
1. Please make sure you have no other security, monitoring, background processes which can offend the FUSE process running. Possibly reproduce under a fresh/clean installation.
2. Please rerun with --debug_fuse --debug_fs --debug_gcs --debug_http --foreground as additional flags to enable debug logs.
3. Monitor the logs and please capture screenshots or copy the relevant logs to a file (can use --log-format and --log-file as well).
4. Attach the screenshot or the logs file to the bug report here.
5. If you're using gcsfuse with any other library/tool/process please list out the steps you took to reproduce the issue.


**System (please complete the following information):**
 - OS: [e.g. Ubuntu 20.04]
 - Platform [VM, Kubernetes]
 - Version [e.g. 0.41]

**Additional context**
Add any other context about the problem here.

**SLO:**
SLO: We strive to respond to all bug reports within 24 business hours during our 
standard business hours of Monday to Friday, 9:00 AM to 5:00 PM [IST Time zone]. 
We will then work diligently to resolve the issue, providing updates and mitigation
steps as available. However, the resolution timeframe may vary depending on the 
complexity of the issue and our development roadmap.
