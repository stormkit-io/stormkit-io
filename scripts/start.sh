#!/usr/bin/env bash
set -e

DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
WORKDIR="$(dirname "${DIR}")"

cd "${WORKDIR}"

if ! command -v overmind &>/dev/null; then
    echo "overmind could not be found"
    exit
fi

if ! command -v tmux &>/dev/null; then
    echo "tmux could not be found"
    exit
fi

# Start dependent services
docker compose up -d db redis

echo "Waiting for database to be ready..."
until docker compose exec db pg_isready -U ${POSTGRES_USER:-stormkit_admin} > /dev/null 2>&1; do
  echo "Database is unavailable - sleeping"
  sleep 2
done

echo "Database is ready!"

echo "Waiting for Redis to be ready..."

until docker compose exec redis redis-cli ping > /dev/null 2>&1; do
  echo "Redis is unavailable - sleeping"
  sleep 2
done

echo "Redis is ready!"
echo "All services are ready, starting application..."
echo "Loading environment variables from from .env file".

if [ -f "$WORKDIR/.env" ]; then
  export $(cat .env | xargs)
fi

export STORMKIT_PROJECT_ROOT=$(pwd)
export GO111MODULE=on
export CGO_ENABLED=1

go mod download

go build -o $WORKDIR/bin/runner $WORKDIR/src/ee/runner

overmind s -f Procfile

