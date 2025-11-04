#!/bin/bash

CONFIG_FILE="test_config.yaml"
# UPDATED: Using 'interrupt' to match the package in the latest execution trace.
PACKAGE_NAME="read_cache"
GO_TEST_DIR="./${PACKAGE_NAME}/..."

# --- 1. Robustness Checks ---
if ! command -v yq &> /dev/null; then
    echo "Error: 'yq' is not installed. Please install it to parse the YAML config." >&2
    exit 1
fi

if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: Configuration file not found at $CONFIG_FILE. Check your current directory." >&2
    exit 1
fi

# Calculate absolute path for the config file.
CONFIG_FILE_ABS="$(pwd)/$CONFIG_FILE"

# --- End Checks ---


# Read base config and details (requires 'yq')
CONFIG_BASE=$(yq ".${PACKAGE_NAME}[0]" "$CONFIG_FILE") 

# Check if CONFIG_BASE returned any data
if [ -z "$CONFIG_BASE" ]; then
    echo "Error: Could not find '${PACKAGE_NAME}[0]' entry in $CONFIG_FILE. Check YAML structure." >&2
    exit 1
fi

MOUNTED_DIR=$(echo "$CONFIG_BASE" | yq -r '.mounted_directory')
TEST_BUCKET=$(echo "$CONFIG_BASE" | yq -r '.test_bucket')
NUM_CONFIGS=$(echo "$CONFIG_BASE" | yq '.configs | length')

# Check if core variables were correctly extracted
if [ "$MOUNTED_DIR" == "null" ] || [ -z "$MOUNTED_DIR" ]; then
    echo "Error: Mounted directory is 'null' or empty. Check the 'mounted_directory' field in your config." >&2
    exit 1
fi
if [ "$NUM_CONFIGS" == "null" ] || [ "$NUM_CONFIGS" -eq 0 ]; then
    echo "Error: Found 0 test configurations. Check the 'configs' array in your config." >&2
    exit 1
fi


echo "Running $NUM_CONFIGS configurations..."

for (( i=0; i<$NUM_CONFIGS; i++ )); do
    # Use 'yq -r' to get the raw string, which will be 'null' if the key doesn't exist.
    TEST_NAME=$(echo "$CONFIG_BASE" | yq -r ".configs[$i].run")
    
    # Extract the number of separate flag sets for this configuration (j loop)
    NUM_FLAG_SETS=$(echo "$CONFIG_BASE" | yq ".configs[$i].flags | length")
    
    # If a specific test name is set, use it, otherwise show the package name
    DISPLAY_NAME=${TEST_NAME}
    if [ "$TEST_NAME" == "null" ] || [ -z "$TEST_NAME" ]; then
        DISPLAY_NAME="All tests in ${PACKAGE_NAME}"
    fi

    # Inner loop iterates over the individual flag sets within the current configuration
    for (( j=0; j<$NUM_FLAG_SETS; j++ )); do
        # Extract the single set of flags for the current iteration
        FLAGS=$(echo "$CONFIG_BASE" | yq -r ".configs[$i].flags[$j]")

        # Conditionally set the RUN_FLAG
        RUN_FLAG=""
        
        # Check if TEST_NAME is not 'null' and not empty
        if [ "$TEST_NAME" != "null" ] && [ ! -z "$TEST_NAME" ]; then
            RUN_FLAG="-run \"^${TEST_NAME}$\""
        fi
        
        echo "--- ${DISPLAY_NAME} (using flags: ${FLAGS}) ---"

        # MOCK MOUNT (This line correctly uses GCSFuse FLAGS)
        echo "  Mount: gcsfuse $TEST_BUCKET $MOUNTED_DIR $FLAGS"
        mkdir -p "$MOUNTED_DIR" 
        
        # RUN TEST
        # Command now correctly includes RUN_FLAG only when defined
        COMMAND="GODEBUG=asyncpreemptoff=1 go test $GO_TEST_DIR -p 1 --integrationTest -v --config-file=\"$CONFIG_FILE_ABS\" $RUN_FLAG"
        echo "  Test: $COMMAND"
        eval $COMMAND # This line executes the command
        
        # MOCK UNMOUNT
        echo "  Unmount: fusermount -u $MOUNTED_DIR"
        rmdir "$MOUNTED_DIR" 2>/dev/null
    done
done