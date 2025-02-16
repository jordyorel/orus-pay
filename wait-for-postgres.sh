#!/bin/sh
# wait-for-postgres.sh

set -e

host="$1"
port="$2"
shift 2  # Remove first two arguments (host/port)
cmd="$@"

# Add connection timeout using PostgreSQL environment variable
until PGCONNECT_TIMEOUT=10 PGPASSWORD=$DB_PASSWORD psql -h "$host" -p "$port" -U "$DB_USER" -d "$DB_NAME" -c '\q'; do
  >&2 echo "Postgres is unavailable - sleeping"
  sleep 1
done

>&2 echo "Postgres is up - executing command"
exec $cmd
