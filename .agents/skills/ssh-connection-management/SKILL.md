---
name: ssh-connection-management
description: Guides on establishing persistent master SSH multiplexing sockets at ~/.ssh/sockets/<TARGET_NAME>.sock using ssh -f -N -M -S, testing connection liveness, gracefully terminating via -O exit, removing stale control sockets, and recreating sockets when remote user permissions or groups change.
---

# SSH Connection Management

This skill guides you through establishing, managing, and troubleshooting persistent SSH socket multiplexing connections to target GCE VMs or remote VMs. Socket multiplexing speeds up command execution and maintains session resilience across automated workflow steps such as remote testing, benchmarking, or environment management.

## Prerequisites & Trigger Conditions

### Prerequisites
1. **GCP Compute SSH Keys**: Local SSH key pair present at `~/.ssh/google_compute_engine` (or standard `~/.ssh/id_rsa`).
2. **GCP Authentication**: Local `gcloud` authenticated with compute viewer/admin permissions.
3. **Network Reachability**: Internal IP or hostname reachability to target VM (e.g. `nic0.<VM_NAME>.<ZONE>.c.<PROJECT_ID>.internal.gcpnode.com` or `<REMOTE_HOST>`).
4. **OpenSSH Client**: Local OpenSSH client supporting ControlMaster options (`-M`, `-S`).

### Trigger Conditions
- Executed prior to running remote commands, setup scripts, conformance test suites, benchmarking, or environment management on target GCE or remote VMs.
- Triggered when SSH connections fail due to stale socket files ("Control socket connect failed").
- Triggered when user permissions or group memberships on target VM are modified requiring session group ID refresh.

## Input/Output Contract

### Inputs
- **Target Connection Details**:
  - Target Name (`<TARGET_NAME>`, e.g., `gce-c4-ssd` or target VM identifier).
  - VM Name (`<VM_NAME>`).
  - Zone (`<ZONE>`).
  - GCP Project ID (`<PROJECT_ID>`).
  - SSH User (`<SSH_USER>`).
- **Socket Cache Directory**: `~/.ssh/sockets/`.

### Outputs
- **Active Master Socket File**: Unix domain socket located at `~/.ssh/sockets/<TARGET_NAME>.sock`.
- **Background SSH Process**: Persistent background `ssh -f -N -M` process holding the master channel open.

## Step-by-Step Procedure

### Step 1: Create Socket Cache Directory

Ensure local socket directory exists:
```bash
mkdir -p ~/.ssh/sockets
```

### Step 2: Clean Up Lingering Sessions and Stale Sockets

Before starting a master connection, gracefully terminate any active master SSH daemon associated with the target to avoid leaving orphaned background processes in memory, and remove any broken or stale socket file:
```bash
ssh -O exit -S ~/.ssh/sockets/<TARGET_NAME>.sock <SSH_USER>@nic0.<VM_NAME>.<ZONE>.c.<PROJECT_ID>.internal.gcpnode.com 2>/dev/null || rm -f ~/.ssh/sockets/<TARGET_NAME>.sock
```

### Step 3: Establish Master SSH Connection

Launch the persistent master SSH connection in the background (forks immediately via `-f`):
```bash
ssh -f -N -M -S ~/.ssh/sockets/<TARGET_NAME>.sock -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/google_compute_engine <SSH_USER>@nic0.<VM_NAME>.<ZONE>.c.<PROJECT_ID>.internal.gcpnode.com
```

Key options explained:
- `-f`: Requests ssh to go to background just before command execution (forks into background).
- `-N`: Do not execute a remote command (background connection mode).
- `-M`: Place the SSH client into master mode for connection sharing.
- `-S ~/.ssh/sockets/<TARGET_NAME>.sock`: Path to the control socket.
- `-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null`: Prevent interactive host key prompts.

### Step 4: Verify Connection Liveness

Test remote command execution over the master socket:
```bash
ssh -S ~/.ssh/sockets/<TARGET_NAME>.sock -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/google_compute_engine <SSH_USER>@nic0.<VM_NAME>.<ZONE>.c.<PROJECT_ID>.internal.gcpnode.com "echo 'Connection Alive'"
```

### Step 5: Refreshing / Recreating Sockets

If user permissions or group memberships change on the remote VM (e.g., after `usermod -aG <group>`):
1. Gracefully terminate the active master SSH session and clean up the socket file:
   ```bash
   ssh -O exit -S ~/.ssh/sockets/<TARGET_NAME>.sock <SSH_USER>@nic0.<VM_NAME>.<ZONE>.c.<PROJECT_ID>.internal.gcpnode.com 2>/dev/null || rm -f ~/.ssh/sockets/<TARGET_NAME>.sock
   ```
2. Re-establish the master connection by repeating Step 3.

## Failure Modes & Edge Cases

| Failure Scenario | Root Cause | Remediation / Recovery Action |
|---|---|---|
| **`Control socket connect failed: Connection refused`** | Master SSH process died unexpectedly, leaving a dead socket file | Gracefully exit or remove stale socket file (`ssh -O exit -S ~/.ssh/sockets/<TARGET_NAME>.sock <SSH_USER>@<HOST> 2>/dev/null || rm -f ~/.ssh/sockets/<TARGET_NAME>.sock`) and re-run master connection command. |
| **`Permission Denied (publickey)`** | SSH key `~/.ssh/google_compute_engine` missing or expired GCP IAM SSH login credentials | Run `gcloud compute config-default-ssh-keys` or `gcloud compute ssh <VM_NAME> --zone=<ZONE>` to refresh SSH keys. |
| **Permission / Group Refresh Delay** | User added to a new group or permissions modified, but commands fail with permission errors | Active SSH master session retains original user and group IDs. Gracefully exit master session (`ssh -O exit -S ~/.ssh/sockets/<TARGET_NAME>.sock <SSH_USER>@<HOST> 2>/dev/null || rm -f ~/.ssh/sockets/<TARGET_NAME>.sock`), and start new master SSH socket. |
| **Connection Drop / Network Disconnect** | Remote VM rebooted or network path reset | Gracefully exit master socket if responsive (`ssh -O exit -S ~/.ssh/sockets/<TARGET_NAME>.sock <SSH_USER>@<HOST> 2>/dev/null || rm -f ~/.ssh/sockets/<TARGET_NAME>.sock`), and re-establish master SSH connection. |

## Verification Checks

1. **Local Socket File Check**: Confirm socket file exists and is an active socket file:
   ```bash
   test -S ~/.ssh/sockets/<TARGET_NAME>.sock && echo "SOCKET_EXISTS"
   ```
2. **Remote Echo Check**: Confirm commands execute over multiplexed socket:
   ```bash
   ssh -S ~/.ssh/sockets/<TARGET_NAME>.sock -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/google_compute_engine <SSH_USER>@nic0.<VM_NAME>.<ZONE>.c.<PROJECT_ID>.internal.gcpnode.com "echo 'Connection Alive'"
   ```
3. **Graceful Teardown Check**: Confirm graceful termination of the master socket:
   ```bash
   ssh -S ~/.ssh/sockets/<TARGET_NAME>.sock -O exit <SSH_USER>@nic0.<VM_NAME>.<ZONE>.c.<PROJECT_ID>.internal.gcpnode.com
   ```
