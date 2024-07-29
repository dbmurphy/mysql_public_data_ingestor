#!/bin/bash
set -e

# Start MySQL server in the background
mysqld &

# Wait for MySQL to be ready
echo "Waiting for MySQL to start..."
until mysql -u root -prootpassword -e "SELECT 1" > /dev/null 2>&1; do
  sleep 1
done

# Execute the initialization SQL script
if [ -f "/docker-entrypoint-initdb.d/init.sql" ]; then
  echo "Running initialization script"
  mysql -u root -prootpassword testdb < /docker-entrypoint-initdb.d/init.sql
else
  echo "Initialization script /docker-entrypoint-initdb.d/init.sql not found!"
  exit 1
fi

# Keep the MySQL server running in the foreground
fg %1
