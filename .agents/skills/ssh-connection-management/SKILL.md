---
name: ssh-connection-management
description: Guides on establishing persistent master SSH multiplexing sockets, optimizing keep-alive, and handling path limits for seamless remote operations.
---

# SSH Connection Management

This skill guides you through establishing, managing, and troubleshooting persistent SSH socket multiplexing connections to target GCE VMs or remote VMs. Socket multiplexing speeds up command execution and maintains session resilience across automated workflow steps such as remote testing, benchmarking, or environment management.

> [!IMPORTANT]
> **Internal Google Infrastructure Note:** Connections to `*.internal.gcpnode.com` run through Google's SUP SSH Relay (`corp-ssh-helper`), which mandates a Gnubby security key touch for **every new connection**. Multiplexing is critical to reduce touches, but requires specific configuration to avoid silent fallbacks or timeouts.

## Optimization Strategies

### 1. Avoid Silent Fallbacks
When using `-S <socket_path>`, if the socket is dead or missing, OpenSSH may silently fall back to a direct connection, triggering a Gnubby touch. Ensure sockets are alive before use.

### 2. Keep-Alive (`ServerAliveInterval`)
UberProxy or intermediate routers may drop idle TCP connections. Always include `ServerAliveInterval` to keep the Master connection alive.

### 3. Socket Path Length Limit
Unix domain sockets have a strict limit of **108 characters**. Lengthy VM names or project IDs can cause creation failure. Use short target names or hash placeholders like `%C`.

---

## 🟢 Recommended: Idiomatic SSH Config + Anchor Session (Option B)

This is the **lowest toil** approach for daily development. It automates multiplexing and guarantees zero surprise touches as long as an "Anchor" is active.

### Step 1: Configure Local SSH (`~/.ssh/config`)
Add this block to your local `~/.ssh/config` (on both your local laptop and remote workstation):

```text
# Optimize connections to Google Internal GCP VMs
Host *.internal.gcpnode.com
    # Default user for these hosts
    User mohitkyadav_google_com
    # Default identity file
    IdentityFile ~/.ssh/google_compute_engine
    
    # Automatically handle connection multiplexing
    ControlMaster auto
    # Use a short hash for the socket path to avoid the 108-character limit
    # CRITICAL: Use Absolute Path if relative paths fail to bind
    ControlPath /usr/local/google/home/mohitkyadav/.ssh/sockets/%C
    # Keep the master connection active for 15 hours
    ControlPersist 15h
    
    # Keep the connection alive (prevent SUP Relay drops)
    ServerAliveInterval 30
    
    # Frictionless connections
    StrictHostKeyChecking accept-new
```

### Step 2: Establish the "Anchor Session"
Open one dedicated terminal and run:
```bash
ssh nic0.<VM_NAME>.<ZONE>.c.<PROJECT_ID>.internal.gcpnode.com "sleep infinity"
```
*(Or simply leave an interactive shell open).*

Touch your Gnubby **once**. As long as this session is alive, all other commands (from scripts, agent, or other terminals) will piggyback on this connection **instantly and with zero touches**.

---

## Alternative: CLI-Based Management (Option A)

Use this for one-off scripts or automation pipelines where you want deterministic control without modifying global configs.

### Step-by-Step Procedure

#### Step 1: Create Socket Cache Directory
Ensure local socket directory exists:
```bash
mkdir -p ~/.ssh/sockets
```

#### Step 2: Clean Up Lingering Sessions and Stale Sockets
Before starting a master connection, gracefully terminate any active master SSH daemon associated with the target to avoid leaving orphaned background processes in memory, and remove any broken or stale socket file:
```bash
ssh -O exit -S ~/.ssh/sockets/<TARGET_NAME>.sock <SSH_USER>@nic0.<VM_NAME>.<ZONE>.c.<PROJECT_ID>.internal.gcpnode.com 2>/dev/null || rm -f ~/.ssh/sockets/<TARGET_NAME>.sock
```

#### Step 3: Establish Master SSH Connection (Optimized)
Launch the persistent master SSH connection in the background with Keep-Alive:
```bash
ssh -f -N -M -S ~/.ssh/sockets/<TARGET_NAME>.sock \
  -o ServerAliveInterval=30 \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \
  -i ~/.ssh/google_compute_engine \
  <SSH_USER>@nic0.<VM_NAME>.<ZONE>.c.<PROJECT_ID>.internal.gcpnode.com
```

#### Step 4: Verify Connection Liveness
Test remote command execution over the master socket:
```bash
ssh -S ~/.ssh/sockets/<TARGET_NAME>.sock -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/google_compute_engine <SSH_USER>@nic0.<VM_NAME>.<ZONE>.c.<PROJECT_ID>.internal.gcpnode.com "echo 'Connection Alive'"
```

---

## Failure Modes & Edge Cases

| Failure Scenario | Root Cause | Remediation / Recovery Action |
|---|---|---|
| **`Control socket connect failed: Connection refused`** | Master SSH process died unexpectedly, leaving a dead socket file | Gracefully exit or remove stale socket file and re-run master connection command. |
| **`Permission Denied (publickey)`** | SSH key `~/.ssh/google_compute_engine` missing or expired GCP IAM SSH login credentials | Run `gcloud compute config-default-ssh-keys` or `gcloud compute ssh <VM_NAME> --zone=<ZONE>` to refresh SSH keys. |
| **Silent Fallback / Repeated Touches** | Dead socket or timeout caused fallback to direct connection | Verify master connection is alive. Ensure `ServerAliveInterval` is set. Check if socket path exceeds 108 chars. |

## Verification Checks (Option A)

1. **Local Socket File Check**: Confirm socket file exists:
   ```bash
   test -S ~/.ssh/sockets/<TARGET_NAME>.sock && echo "SOCKET_EXISTS"
   ```
2. **Remote Echo Check**: Confirm commands execute over multiplexed socket:
   ```bash
   ssh -S ~/.ssh/sockets/<TARGET_NAME>.sock -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/google_compute_engine <SSH_USER>@nic0.<VM_NAME>.<ZONE>.c.<PROJECT_ID>.internal.gcpnode.com "echo 'Connection Alive'"
   ```
