config=$(jq --arg EXPERIMENT_NUMBER "$EXPERIMENT_NUMBER" --arg current_date "$current_date" '.experiment_configuration[] | select(.end_date >= $current_date)' test.json | jq -s ".[$EXPERIMENT_NUMBER-1]")
# Access specific properties of the configuration
CONFIG_NAME=$(echo "$config" | jq -r '.config_name')
GCSFUSE_FLAGS=$(echo "$config" | jq -r '.gcsfuse_flags')
BRANCH=$(echo "$config" | jq -r '.branch')
END_DATE=$(echo "$config" | jq -r '.end_date')
# Print the config_file json
echo "$config_file_json"
