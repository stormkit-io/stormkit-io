#!/usr/bin/env bash
set -e

OVERMIND_RELEASE_URL="https://github.com/DarthSim/overmind/releases/latest"
OVERMIND_RELEASE_DOWNLOAD_URL="https://github.com/DarthSim/overmind/releases/download"
OVERMIND_RELEASE_TAG=""

is_procfile_overmind() {
  local candidate="$1"

  "$candidate" help start >/dev/null 2>&1
}

download_procfile_overmind() {
  local os arch asset url target tmpfile

  case "$(uname -s)" in
    Linux) os="linux" ;;
    *)
      echo "Automatic download is only supported on Linux. Please install Overmind manually from ${OVERMIND_RELEASE_URL}."
      return 1
      ;;
  esac

  case "$(uname -m)" in
    x86_64 | amd64) arch="amd64" ;;
    arm64 | aarch64) arch="arm64" ;;
    *)
      echo "Unsupported CPU architecture $(uname -m). Please install Overmind manually from ${OVERMIND_RELEASE_URL}."
      return 1
      ;;
  esac

  if ! command -v curl >/dev/null 2>&1; then
    echo "curl is required to download Overmind automatically. Install curl or download Overmind manually from ${OVERMIND_RELEASE_URL}."
    return 1
  fi

  if ! command -v gzip >/dev/null 2>&1; then
    echo "gzip is required to extract the Overmind archive. Install gzip or download Overmind manually from ${OVERMIND_RELEASE_URL}."
    return 1
  fi

  if [ -z "$OVERMIND_RELEASE_TAG" ]; then
    if ! OVERMIND_RELEASE_TAG="$(get_latest_overmind_release_tag)"; then
      echo "Unable to determine latest Overmind release. Please install it manually from ${OVERMIND_RELEASE_URL}."
      return 1
    fi
  fi

  asset="overmind-${OVERMIND_RELEASE_TAG}-${os}-${arch}.gz"
  url="${OVERMIND_RELEASE_DOWNLOAD_URL}/${OVERMIND_RELEASE_TAG}/${asset}"
  target="$WORKDIR/bin/overmind"
  mkdir -p "$(dirname "$target")"
  tmpfile="$(mktemp)"

  echo "Downloading ${asset}..."
  if ! curl -L --fail --silent --show-error "$url" -o "$tmpfile"; then
    echo "Failed to download ${asset} from ${url}"
    rm -f "$tmpfile"
    return 1
  fi

  if ! gzip -d -c "$tmpfile" >"$target"; then
    echo "Failed to extract Overmind archive."
    rm -f "$tmpfile"
    return 1
  fi

  rm -f "$tmpfile"
  chmod +x "$target"
  OVERMIND_CMD="$target"
  echo "Procfile Overmind installed locally at $target"
  return 0
}

get_latest_overmind_release_tag() {
  local final_url

  if ! command -v curl >/dev/null 2>&1; then
    return 1
  fi

  final_url="$(curl -Ls -o /dev/null -w '%{url_effective}' "${OVERMIND_RELEASE_URL}")" || return 1
  final_url="${final_url%/}"
  basename "$final_url"
}

ensure_procfile_overmind() {
  local candidate

  if [ -n "$OVERMIND_BIN" ]; then
    candidate="$OVERMIND_BIN"

    if [ ! -x "$candidate" ]; then
      echo "OVERMIND_BIN is set to '$candidate' but it is not executable."
      exit 1
    fi

    if is_procfile_overmind "$candidate"; then
      OVERMIND_CMD="$candidate"
      return
    fi

    cat <<EOF
The binary specified by OVERMIND_BIN ($candidate) does not expose Procfile commands.
Download the Procfile release from ${OVERMIND_RELEASE_URL}, make it executable, and point OVERMIND_BIN to it.
EOF
    exit 1
  fi

  if command -v overmind &>/dev/null; then
    candidate="$(command -v overmind)"
    if is_procfile_overmind "$candidate"; then
      OVERMIND_CMD="$candidate"
      return
    fi

    echo "Detected an overmind binary at $candidate, but it appears to be the SaaS CLI (no Procfile support)."
    echo "Downloading the Procfile binary locally..."
    if download_procfile_overmind; then
      return
    fi

    cat <<EOF
Automatic download failed. Install the Procfile version from ${OVERMIND_RELEASE_URL} and ensure it is earlier in your PATH, or set OVERMIND_BIN to that binary.
EOF
    exit 1
  fi

  echo "Procfile Overmind binary not found in PATH. Attempting to download it locally..."
  if download_procfile_overmind; then
    return
  fi

  cat <<EOF
Automatic download failed. Download the Procfile release from ${OVERMIND_RELEASE_URL} and ensure it is available in PATH, or set OVERMIND_BIN to the binary path.
EOF
  exit 1
}

install_tmux_with_apt() {
  local cmd

  if ! command -v apt-get >/dev/null 2>&1; then
    return 1
  fi

  if [ "$EUID" -eq 0 ]; then
    cmd=(apt-get)
  elif command -v sudo >/dev/null 2>&1 && sudo -n true >/dev/null 2>&1; then
    cmd=(sudo apt-get)
  else
    return 1
  fi

  echo "tmux is not installed. Installing via apt-get..."
  if ! "${cmd[@]}" update; then
    echo "apt-get update failed"
    return 1
  fi

  if ! "${cmd[@]}" install -y tmux; then
    echo "apt-get install tmux failed"
    return 1
  fi

  return 0
}

ensure_tmux() {
  if command -v tmux >/dev/null 2>&1; then
    return
  fi

  if install_tmux_with_apt; then
    return
  fi

  cat <<EOF
tmux could not be found. Install tmux (e.g. `sudo apt install tmux`) before running this script.
EOF
  exit 1
}

ensure_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    cat <<'EOF'
Docker CLI could not be found. Install Docker Engine and the Compose plugin before running this script.
On Debian/Ubuntu:
  sudo apt update
  sudo apt install docker.io docker-compose-plugin

After installation ensure your user is part of the `docker` group or run the script with sudo.
See https://docs.docker.com/engine/install/ for other distributions.
EOF
    exit 1
  fi

  if docker compose version >/dev/null 2>&1; then
    return
  fi

  if command -v docker-compose >/dev/null 2>&1; then
    cat <<'EOF'
Found legacy docker-compose binary but `docker compose` plugin is missing. Install the Compose V2 plugin or update Docker to a version that includes it.
See https://docs.docker.com/compose/install/ for installation instructions.
EOF
  else
    cat <<'EOF'
Docker Compose V2 plugin (`docker compose`) could not be found. Install the Compose plugin before running this script.
See https://docs.docker.com/compose/install/ for installation instructions.
EOF
  fi
  exit 1
}

DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
WORKDIR="$(dirname "${DIR}")"

cd "${WORKDIR}"
if [ ! -f "$WORKDIR/.env" ] && [ -f "$WORKDIR/.env.example" ]; then
  echo "Creating .env from .env.example"
  cp "$WORKDIR/.env.example" "$WORKDIR/.env"
fi

ensure_procfile_overmind

ensure_tmux
ensure_docker

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

"$OVERMIND_CMD" s -f Procfile
