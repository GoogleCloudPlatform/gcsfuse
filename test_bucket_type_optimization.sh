#!/bin/bash
# Integration test script for bucket-type-based FUSE optimizations
# This script helps test the feature with real GCS buckets

set -e

GCSFUSE_BIN="${GCSFUSE_BIN:-./gcsfuse}"
MOUNT_POINT="${MOUNT_POINT:-/tmp/gcsfuse-test-mount}"
LOG_FILE="/tmp/gcsfuse-bucket-type-test.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    if mountpoint -q "$MOUNT_POINT" 2>/dev/null; then
        log_info "Unmounting $MOUNT_POINT"
        fusermount -u "$MOUNT_POINT" || umount "$MOUNT_POINT" || true
    fi
    rm -f "$LOG_FILE"
}

# Trap to cleanup on exit
trap cleanup EXIT

# Check if gcsfuse binary exists
if [ ! -f "$GCSFUSE_BIN" ]; then
    log_error "GCSFuse binary not found at $GCSFUSE_BIN"
    log_info "Please build gcsfuse first: go build ."
    exit 1
fi

# Create mount point if it doesn't exist
mkdir -p "$MOUNT_POINT"

# Test function
test_bucket_type_optimization() {
    local bucket_name=$1
    local expected_type=$2
    local test_name=$3
    
    log_info "=========================================="
    log_info "Test: $test_name"
    log_info "Bucket: $bucket_name"
    log_info "Expected Type: $expected_type"
    log_info "=========================================="
    
    # Mount with trace logging
    log_info "Mounting bucket..."
    "$GCSFUSE_BIN" \
        --log-severity=trace \
        --log-file="$LOG_FILE" \
        --foreground=false \
        "$bucket_name" \
        "$MOUNT_POINT" 2>&1 | tee /tmp/mount-output.log &
    
    MOUNT_PID=$!
    
    # Wait for mount to complete
    sleep 3
    
    # Check if mount was successful
    if ! mountpoint -q "$MOUNT_POINT"; then
        log_error "Mount failed!"
        kill $MOUNT_PID 2>/dev/null || true
        cat /tmp/mount-output.log
        return 1
    fi
    
    log_info "Mount successful!"
    
    # Check log for bucket type detection
    if grep -q "Detected bucket type: $expected_type" "$LOG_FILE"; then
        log_info "✓ Bucket type correctly detected as: $expected_type"
    else
        log_warn "Bucket type detection message not found in logs"
        log_info "Searching for bucket type in logs..."
        grep -i "bucket.*type" "$LOG_FILE" || log_warn "No bucket type messages found"
    fi
    
    # Check for optimization messages
    if [ "$expected_type" = "zonal" ]; then
        if grep -q "Applied.*bucket-type-based optimizations" "$LOG_FILE"; then
            log_info "✓ Bucket-type optimizations were applied"
            
            # Check specific optimizations
            log_info "Checking applied optimizations:"
            
            if grep -q "file-system.max-background.*128" "$LOG_FILE"; then
                log_info "  ✓ max-background = 128"
            else
                log_warn "  ? max-background optimization not found in logs"
            fi
            
            if grep -q "file-system.congestion-threshold.*96" "$LOG_FILE"; then
                log_info "  ✓ congestion-threshold = 96"
            else
                log_warn "  ? congestion-threshold optimization not found in logs"
            fi
            
            if grep -q "file-system.async-read.*true" "$LOG_FILE"; then
                log_info "  ✓ async-read = true"
            else
                log_warn "  ? async-read optimization not found in logs"
            fi
        else
            log_warn "No optimization messages found in logs"
        fi
        
        # Check mount options
        log_info "Checking actual mount options..."
        mount | grep "$MOUNT_POINT" | grep -o "max_background=[0-9]*" || log_warn "max_background not in mount options"
        mount | grep "$MOUNT_POINT" | grep -o "congestion_threshold=[0-9]*" || log_warn "congestion_threshold not in mount options"
        mount | grep "$MOUNT_POINT" | grep -o "async_read" || log_warn "async_read not in mount options"
    else
        log_info "Non-zonal bucket - checking that no optimizations were applied"
        if ! grep -q "Applied.*bucket-type-based optimizations" "$LOG_FILE"; then
            log_info "✓ Correctly skipped optimizations for $expected_type bucket"
        else
            log_warn "Unexpected: optimizations were applied to $expected_type bucket"
        fi
    fi
    
    # Unmount
    log_info "Unmounting..."
    fusermount -u "$MOUNT_POINT" || umount "$MOUNT_POINT"
    sleep 1
    
    log_info "Test completed!"
    echo ""
}

# Test with user-provided flags
test_user_override() {
    local bucket_name=$1
    
    log_info "=========================================="
    log_info "Test: User Override"
    log_info "Bucket: $bucket_name"
    log_info "Testing that user values take precedence"
    log_info "=========================================="
    
    # Mount with user-specified values
    log_info "Mounting with --max-background=512..."
    "$GCSFUSE_BIN" \
        --log-severity=trace \
        --log-file="$LOG_FILE" \
        --max-background=512 \
        --foreground=false \
        "$bucket_name" \
        "$MOUNT_POINT" 2>&1 &
    
    MOUNT_PID=$!
    sleep 3
    
    if ! mountpoint -q "$MOUNT_POINT"; then
        log_error "Mount failed!"
        kill $MOUNT_PID 2>/dev/null || true
        return 1
    fi
    
    # Check that user value is used
    if mount | grep "$MOUNT_POINT" | grep -q "max_background=512"; then
        log_info "✓ User value (512) correctly used instead of optimization (128)"
    else
        log_warn "Could not verify user value in mount options"
        mount | grep "$MOUNT_POINT"
    fi
    
    fusermount -u "$MOUNT_POINT" || umount "$MOUNT_POINT"
    sleep 1
    
    log_info "Test completed!"
    echo ""
}

# Test with disabled autoconfig
test_disabled_autoconfig() {
    local bucket_name=$1
    
    log_info "=========================================="
    log_info "Test: Disabled Autoconfig"
    log_info "Bucket: $bucket_name"
    log_info "Testing with --disable-autoconfig"
    log_info "=========================================="
    
    "$GCSFUSE_BIN" \
        --log-severity=trace \
        --log-file="$LOG_FILE" \
        --disable-autoconfig \
        --foreground=false \
        "$bucket_name" \
        "$MOUNT_POINT" 2>&1 &
    
    MOUNT_PID=$!
    sleep 3
    
    if ! mountpoint -q "$MOUNT_POINT"; then
        log_error "Mount failed!"
        kill $MOUNT_PID 2>/dev/null || true
        return 1
    fi
    
    # Check that no optimizations were applied
    if ! grep -q "Applied.*bucket-type-based optimizations" "$LOG_FILE"; then
        log_info "✓ Correctly skipped optimizations with --disable-autoconfig"
    else
        log_warn "Unexpected: optimizations were applied despite --disable-autoconfig"
    fi
    
    fusermount -u "$MOUNT_POINT" || umount "$MOUNT_POINT"
    sleep 1
    
    log_info "Test completed!"
    echo ""
}

# Main execution
main() {
    log_info "Starting Bucket-Type FUSE Optimization Tests"
    log_info "=============================================="
    echo ""
    
    if [ $# -lt 1 ]; then
        log_error "Usage: $0 <bucket-name> [bucket-type]"
        log_info ""
        log_info "Examples:"
        log_info "  $0 my-zonal-bucket zonal"
        log_info "  $0 my-hns-bucket hierarchical"
        log_info "  $0 my-standard-bucket standard"
        log_info ""
        log_info "If bucket-type is not provided, tests will run but won't verify the type"
        exit 1
    fi
    
    BUCKET_NAME=$1
    BUCKET_TYPE=${2:-"unknown"}
    
    log_info "Using bucket: $BUCKET_NAME"
    log_info "Expected type: $BUCKET_TYPE"
    echo ""
    
    # Run tests
    test_bucket_type_optimization "$BUCKET_NAME" "$BUCKET_TYPE" "Basic mount with auto-optimization"
    
    if [ "$BUCKET_TYPE" = "zonal" ]; then
        test_user_override "$BUCKET_NAME"
        test_disabled_autoconfig "$BUCKET_NAME"
    fi
    
    log_info "=============================================="
    log_info "All tests completed!"
    log_info "=============================================="
    log_info ""
    log_info "Log file: $LOG_FILE"
    log_info "You can review detailed logs with: less $LOG_FILE"
}

main "$@"
