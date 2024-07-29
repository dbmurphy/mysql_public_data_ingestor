#!/bin/bash

set -e

# Variables
IMAGE_NAME="mysql_public_data_ingestor"
INIT_SQL_PATH="./docker/init.sql"
TEMP_SQL_PATH="./docker/temp_init.sql"
CONFIG_FILE="./config-test.yaml"

# Function to read plugin name from config-test.yaml
get_plugin_name() {
  # Use yq or a similar tool to extract the plugin name from config-test.yaml
  # You might need to install yq: https://github.com/mikefarah/yq
  PLUGIN_NAME=$(yq eval '.plugin' $CONFIG_FILE)
  if [ -z "$PLUGIN_NAME" ]; then
    echo "No plugin name found in $CONFIG_FILE!"
    exit 1
  fi
  echo $PLUGIN_NAME
}

# Function to check if init.sql has changed
check_init_sql() {
  PLUGIN_NAME=$(get_plugin_name)
  if [ -z "$PLUGIN_NAME" ]; then
    exit 1
  fi

  PLUGIN_INIT_SQL="./api_plugins/${PLUGIN_NAME}/acc_init.sql"
  if [ ! -f "$PLUGIN_INIT_SQL" ]; then
    echo "$PLUGIN_INIT_SQL does not exist!"
    exit 1
  fi

  cp "$PLUGIN_INIT_SQL" "$TEMP_SQL_PATH"

  # Compare with existing image's init.sql if exists
  if [ -f "$INIT_SQL_PATH" ]; then
    if ! diff "$INIT_SQL_PATH" "$TEMP_SQL_PATH" > /dev/null; then
      return 1
    fi
  else
    # Assume false if init.sql does not exist
    return 1
  fi
}

# Function to clean up old Docker images
cleanup_docker() {
  if docker image inspect $IMAGE_NAME > /dev/null 2>&1; then
    echo "Cleaning up old Docker image..."
    docker rmi -f $IMAGE_NAME || true
  else
    echo "No existing Docker image found."
  fi
}

# Function to build Docker image
build_docker() {
  echo "Copying init.sql to Docker context..."
  cp "$TEMP_SQL_PATH" "$INIT_SQL_PATH"
  echo "Building Docker image..."
  docker build -t $IMAGE_NAME .
}

# Main script
if check_init_sql; then
  echo "No changes detected in init.sql. Skipping build."
  exit 0
else
  cleanup_docker
  build_docker
fi
