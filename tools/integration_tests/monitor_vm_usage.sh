#!/bin/bash
# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# --- Helper function to get CPU Usage ---
# Outputs: CPU Usage percentage (numeric, e.g., "56")
# Returns: 0 on success, 1 on failure (and echoes "0")
get_cpu_usage() {
    local numeric_cpu_usage="0"
    local status=1
    local cpu_idle_raw
    
    # Capture output and check command success
    cpu_idle_raw=$(LC_ALL=C top -bn1 | grep '^%Cpu' | awk '{print $8}' 2>/dev/null)
    local command_status=$?

    if [ "$command_status" -eq 0 ] && [[ "$cpu_idle_raw" =~ ^[0-9.]+$ ]]; then
        numeric_cpu_usage=$(awk -v idle="$cpu_idle_raw" 'BEGIN {printf "%.0f", 100 - idle}')
        # Check if awk succeeded in producing a number
        if [[ "$numeric_cpu_usage" =~ ^[0-9]+$ ]]; then
            status=0
        else
            numeric_cpu_usage="0" # Reset if awk output wasn't as expected
        fi
    fi
    echo "$numeric_cpu_usage"
    return "$status"
}

# --- Helper function to get Memory Usage ---
# Outputs: Memory Usage percentage (numeric, e.g., "41")
# Returns: 0 on success, 1 on failure (and echoes "0")
get_mem_usage() {
    local numeric_mem_percentage="0"
    local status=1
    local mem_info_line
    
    mem_info_line=$(LC_ALL=C free -m | awk '/^Mem:/' 2>/dev/null)
    local command_status=$?

    if [ "$command_status" -eq 0 ] && [ -n "$mem_info_line" ]; then
        local parsed_used parsed_total
        parsed_used=$(echo "$mem_info_line" | awk '{print $3}')
        parsed_total=$(echo "$mem_info_line" | awk '{print $2}')
        
        if [[ "$parsed_total" =~ ^[0-9]+$ ]] && [ "$parsed_total" -gt 0 ] && \
            [[ "$parsed_used" =~ ^[0-9]+$ ]]; then
            numeric_mem_percentage=$(awk -v used="$parsed_used" -v total="$parsed_total" 'BEGIN {printf "%.0f", (used/total)*100}')
            if [[ "$numeric_mem_percentage" =~ ^[0-9]+$ ]]; then
                status=0
            else
                numeric_mem_percentage="0"
            fi
        fi
    fi
    echo "$numeric_mem_percentage"
    return "$status"
}

# --- Helper function to get Disk Usage for / (root filesystem) ---
# Outputs: Disk Usage percentage (numeric, e.g., "44")
# Returns: 0 on success, 1 on failure (and echoes "0")
get_disk_usage() {
    local numeric_disk_percentage="0"
    local status=1
    local disk_info_root_line

    disk_info_root_line=$(LC_ALL=C df -Pk / | awk 'NR==2' 2>/dev/null)
    local command_status=$?

    if [ "$command_status" -eq 0 ] && [ -n "$disk_info_root_line" ]; then
        local raw_percentage
        raw_percentage=$(echo "$disk_info_root_line" | awk '{print $5}')
        
        if [[ "$raw_percentage" =~ ^[0-9]+%$ ]]; then # Checks for format like "56%"
            numeric_disk_percentage=${raw_percentage%\%} # Removes %
            status=0
        fi
    fi
    echo "$numeric_disk_percentage"
    return "$status"
}

update_line() {
    local filepath="$1"
    local line_number_raw="$2"
    local text_to_append="$3"
    local prefix="$4"
    local formatted_line_number

    formatted_line_number=$(printf "%02d" "${line_number_raw}")

    sed -i "/^${prefix}@${formatted_line_number}/s/.*/&${text_to_append}/" "${filepath}"
}

FILEPATH="$1"
INTERVAL=10

echo "CPU UAGE EVERY ${INTERVAL} SECONDS" >> "$FILEPATH"

for i in {1..100}; do
    line=$((100 - $i))
    printf "CPU@%02d|\n" "$line" >> "$FILEPATH"
done

echo "" >> "$FILEPATH"
echo "MEM USAGE EVERY ${INTERVAL} SECONDS" >> "$FILEPATH"

for i in {1..100}; do
    line=$((100 - $i))
    printf "MEM@%02d|\n" "$line" >> "$FILEPATH"
done

echo "" >> "$FILEPATH"
echo "DISK USAGE EVERY ${INTERVAL} SECONDS" >> "$FILEPATH"

for i in {1..100}; do
    line=$((100 - $i))
    printf "DISK@%02d|\n" "$line" >> "$FILEPATH"
done
monitor="true"

clean() {
    monitor="false"
    sleep 20 # Give enough time for file to be updated.
}

trap clean SIGINT SIGTERM

while $monitor; do
    cpu=$(get_cpu_usage)
    mem=$(get_mem_usage)
    disk=$(get_disk_usage)
    for i in {1..100}
    do
        c="*"
        m="*"
        d="*"
        if [[ $i -ge $cpu ]]; then
            c=" "
        fi
        if [[ $i -ge $mem ]]; then
            m=" "
        fi
        if [[ $i -ge $disk ]]; then
            d=" "
        fi
        update_line "$FILEPATH" "$i" "$c" "CPU"
        update_line "$FILEPATH" "$i" "$m" "MEM"
        update_line "$FILEPATH" "$i" "$d" "DISK"
    done
    sleep $INTERVAL
done