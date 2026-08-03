#!/bin/sh

# curl -sSL https://www.stormkit.io/install.sh | sh

# Function to check if a command exists
command_exists() {
  command -v "$@" >/dev/null 2>&1
}

# Agent mode installs Stormkit non-interactively and provisions an owner
# admin + API key so an agent can manage the instance over MCP right away.
#
#   curl -sSL https://www.stormkit.io/install.sh | sh -s -- --agent
#
# Optional: --domain <domain> to skip the auto-generated domain, and
# --email <email> to override the default agent@<domain> admin email.
#
# On an instance that already has users, bootstrap is a no-op and no key is
# provisioned. Do NOT pass --email matching an existing user of an instance
# first set up via OAuth/magic-link: on a single-user instance that collision
# lets bootstrap graft the owner key + a password login onto that account.
AGENT_MODE="0"
CUSTOM_DOMAIN_ARG=""
ADMIN_EMAIL_ARG=""

while [ $# -gt 0 ]; do
  case "$1" in
    --agent) AGENT_MODE="1" ;;
    --domain) CUSTOM_DOMAIN_ARG="$2"; shift ;;
    --domain=*) CUSTOM_DOMAIN_ARG="${1#*=}" ;;
    --email) ADMIN_EMAIL_ARG="$2"; shift ;;
    --email=*) ADMIN_EMAIL_ARG="${1#*=}" ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
  shift
done

IS_MAC="0"

if [ "$(uname -s | cut -c1-6)" = "Darwin" ]; then
  IS_MAC="1"

  if command_exists brew; then
    echo "Brew already installed"
  else
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  fi

  if command_exists docker; then
    echo "Docker already installed"
  else
    brew install docker
  fi

  # Check if Docker is running
  docker_stats=$(docker stats --no-stream)

  if docker info > /dev/null 2>&1; then
    echo "Docker is already running."
  else
    echo "Docker is not running. Starting Docker..."

    # Start Docker
    open --background -a Docker

    # Wait until Docker is running
    while ! docker info >/dev/null 2>&1; do
      echo "Waiting for Docker to start..."
      sleep 2
    done

    echo "Docker has started."
  fi
elif [ "$(uname -s | cut -c1-5)" = "Linux" ]; then
  # We have to run as root
  if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root" >&2
    exit 1
  fi

  # Check if ports 80 or 443 are in use
  for port in 80 443; do
    if ss -tulnp | grep ":${port} " >/dev/null; then
      echo "Port ${port} is already in use" >&2
      exit 1
    fi
  done

  # Install Docker if not already installed
  if command_exists docker; then
    echo "Docker already installed"
  else
    if grep -q "Rocky" /etc/os-release; then
      # See sudo https://docs.rockylinux.org/10/gemstones/containers/docker/
      sudo dnf -y install dnf-plugins-core
      sudo dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
      sudo dnf -y install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
      sudo systemctl start docker
    elif grep -q "Fedora" /etc/os-release; then
      # See https://docs.docker.com/engine/install/fedora/
      sudo dnf -y install dnf-plugins-core
      sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
      sudo dnf -y install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
      sudo systemctl start docker
    else
      curl -sSL https://get.docker.com | sh
    fi

  fi
fi

# Define text colors for output
BLUE="\033[0;34m"
GREEN="\033[0;32m"
PURPLE="\033[0;35m"
GRAY="\033[1;30m"
NC="\033[0m"

LAST_VAR=""

update_env_var_in_env_file() {
  var_name=$1
  var_value=$2

  # Check if the variable already exists in the .env file
  if grep -q "^$var_name=" .env; then
    # Update the existing variable
    if [ "$IS_MAC" = "1" ]; then
      sed -i '' "s/^$var_name=.*/$var_name=$var_value/" .env
    else
      if [ "$var_value" = "''" ]; then
        # If the value is empty, remove the variable
        sed -i~ "/^$var_name=/d" .env
      else
        # Update the variable with the new value
        # Use sed to replace the line with the new value
        # The ~ suffix creates a backup file
        sed -i~ "/^$var_name=/s/=.*/=\"$var_value\"/" .env
      fi
    fi
  else
    # Append the new variable to the .env file
    echo "$var_name=$var_value" >> .env
  fi

  # Remove the swap file (if any)
  rm -rf .env~
}

# Function to prompt for an environment variable and update the .env file
update_env_var() {
  var_name=$1
  prompt_message=$2
  prompt_desc=$3
  default_value=$4

  # Prompt for the value of the variable
  if [ -n "$prompt_desc" ]; then
    printf "${PURPLE}%s ${NC}\n${GRAY}%s: ${NC}" "$prompt_message" "$prompt_desc"
  else
    printf "${PURPLE}%s: ${NC}" "$prompt_message"
  fi

  read -r var_value </dev/tty

  # Check if .env file exists, if not, create it
  if [ ! -f .env ]; then
    touch .env
  fi

  if [ -n "$default_value" ]; then
    if [ -z "$var_value" ]; then
      var_value=$default_value
    fi
  fi

  update_env_var_in_env_file $var_name "$var_value"

  LAST_VAR=$var_value
}

SELECTED_PROVIDER=""

single_select() {
  prompt_message=$1
  options=$2 # Define options as a space-separated string
  desc=${3:-"Select an option"}

  printf "${PURPLE}%s${NC}\n" "$prompt_message"
  printf "${GRAY}%s:${NC}\n" "$desc"

  while true; do
    # Use a counter to display the options
    i=1
    for option in $options; do
      printf "%d) %s\n" "$i" "$option"
      i=$((i + 1))
    done

    # Prompt for user input
    echo
    printf "Enter the number of the provider (and press Enter): "
    read -r choice </dev/tty

    # Validate the choice (ensure it's a number and in range)
    if [ "$choice" -ge 1 ] 2>/dev/null && [ "$choice" -le $((i - 1)) ]; then
      # Get the selected provider based on choice
      i=1
      for option in $options; do
        if [ "$i" -eq "$choice" ]; then
          SELECTED_PROVIDER="$option"
          break
        fi
        i=$((i + 1))
      done
      break
    else
      echo "Invalid choice. Please try again."
    fi
  done
}

# Generates a random alphanumeric string. $1 = openssl byte length to draw
# from, $2 = number of characters to keep.
rand_alnum() {
  openssl rand -base64 "$1" | tr -dc 'a-zA-Z0-9' | head -c "$2"
}

# Returns 0 when $1 is a syntactically valid hostname (<= 253 chars).
is_valid_domain() {
  echo "$1" | grep -qE '^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$' &&
    [ ${#1} -le 253 ]
}

setup_base_env_variables() {
  # Download the example .env file
  curl -o ".env" "https://raw.githubusercontent.com/stormkit-io/stormkit-io/main/deploy/.env.example" --silent

  update_env_var_in_env_file POSTGRES_PASSWORD $(rand_alnum 12 24)
  update_env_var_in_env_file STORMKIT_APP_SECRET $(rand_alnum 48 32)
  update_env_var_in_env_file GRAFANA_ADMIN_PASSWORD $(rand_alnum 12 24)
}

# Downloads the Prometheus and Grafana configuration.
#
# These are always fetched, even though nothing here runs by default: the
# `monitoring` compose profile is what decides that. Having the files on disk
# means turning monitoring on later is a flag flip rather than a download.
#
# Keep this list in sync with deploy/monitoring/ in the repository.
setup_monitoring_configs() {
  base="https://raw.githubusercontent.com/stormkit-io/stormkit-io/main/deploy/monitoring"

  mkdir -p monitoring/grafana/provisioning/datasources \
    monitoring/grafana/provisioning/dashboards \
    monitoring/grafana/dashboards

  curl -o "monitoring/prometheus.yml" "$base/prometheus.yml" --silent
  curl -o "monitoring/grafana/provisioning/datasources/prometheus.yml" "$base/grafana/provisioning/datasources/prometheus.yml" --silent
  curl -o "monitoring/grafana/provisioning/dashboards/dashboards.yml" "$base/grafana/provisioning/dashboards/dashboards.yml" --silent
  curl -o "monitoring/grafana/dashboards/stormkit-host.json" "$base/grafana/dashboards/stormkit-host.json" --silent
  curl -o "monitoring/grafana/dashboards/stormkit-dependencies.json" "$base/grafana/dashboards/stormkit-dependencies.json" --silent
}

DOMAIN=""

# Setup the environment variable for the Hosting Service.
setup_domain() {
  # In agent mode there is no TTY to prompt on: honour --domain if given,
  # otherwise fall through to the auto-generated sslip.io domain below.
  if [ "$AGENT_MODE" = "1" ]; then
    if [ -n "$CUSTOM_DOMAIN_ARG" ]; then
      if ! is_valid_domain "$CUSTOM_DOMAIN_ARG"; then
        printf "${GRAY}Invalid --domain value: %s${NC}\n" "$CUSTOM_DOMAIN_ARG" >&2
        exit 1
      fi

      DOMAIN="$CUSTOM_DOMAIN_ARG"
      printf "${GREEN}Using custom domain: $DOMAIN${NC}\n"
      update_env_var_in_env_file STORMKIT_DOMAIN "$DOMAIN"
      return
    fi
  else
  while true; do
    # Ask the user if they have a custom domain
    printf "${PURPLE}Do you have a custom domain you'd like to use?${NC}\n"
    printf "${GRAY}Enter your domain (e.g., myapp.example.com) or press Enter to use auto-generated domain: ${NC}"
    read -r custom_domain </dev/tty

    if [ -z "$custom_domain" ]; then
      # User pressed Enter without entering a domain, use auto-generated
      break
    fi

    # Validate the domain format
    if echo "$custom_domain" | grep -qE '^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$'; then
      # Check if domain length is reasonable (max 253 characters)
      if [ ${#custom_domain} -le 253 ]; then
        DOMAIN="$custom_domain"
        printf "${GREEN}Using custom domain: $DOMAIN${NC}\n"
        update_env_var_in_env_file STORMKIT_DOMAIN "$DOMAIN"
        return
      else
        printf "${GRAY}Domain too long (max 253 characters). Please try again.${NC}\n"
        echo
      fi
    else
      printf "${GRAY}Invalid domain format. Please enter a valid domain (e.g., myapp.example.com) or press Enter to skip.${NC}\n"
      echo
    fi
  done
  fi

  # Fallback to IP-based domain
  printf "${GRAY}Generating auto-generated domain...${NC}\n"

  IP4=$(curl -s -4 ifconfig.me | tr '.' '-')

  # Check if IP4 is a valid IP format (4 octets, each 0-255)
  if ! echo "$IP4" | grep -qE '^([0-9]{1,3}-){3}[0-9]{1,3}$'; then
    IP4=$(curl -s http://checkip.dyndns.org/ | grep -o "[[:digit:].]\+" | tr '.' '-')
  fi

  DOMAIN="$IP4.sslip.io"
  printf "${GREEN}Using auto-generated domain: $DOMAIN${NC}\n"

  update_env_var_in_env_file STORMKIT_DOMAIN "$DOMAIN"
}

AGENT_API_KEY=""
ADMIN_EMAIL=""

# Provisions the credentials the server reads on first boot to create an
# owner admin and mint a matching API key. The key is generated here so the
# raw value never has to be read back out of the container; the server only
# ever stores its hash.
setup_agent_env() {
  ADMIN_EMAIL="$ADMIN_EMAIL_ARG"

  if [ -z "$ADMIN_EMAIL" ]; then
    ADMIN_EMAIL="agent@$DOMAIN"
  fi

  ADMIN_PASSWORD=$(rand_alnum 24 24)
  AGENT_API_KEY="SK_$(openssl rand -hex 31)"

  update_env_var_in_env_file STORMKIT_ADMIN_EMAIL "$ADMIN_EMAIL"
  update_env_var_in_env_file STORMKIT_ADMIN_PASSWORD "$ADMIN_PASSWORD"
  update_env_var_in_env_file STORMKIT_AGENT_API_KEY "$AGENT_API_KEY"
}

# For now keep this file here - we need to improve the docker swarm setup
# Download the docker-compose.yaml file
curl -o "docker-compose.yaml" "https://raw.githubusercontent.com/stormkit-io/stormkit-io/main/deploy/docker-compose.yaml" --silent

setup_base_env_variables
setup_monitoring_configs
setup_domain

if [ "$AGENT_MODE" = "1" ]; then
  setup_agent_env
fi

docker compose up -d

echo ""
printf "${GREEN}Congratulations, Stormkit is installed on your computer!\n${NC}"

# Function to wait for the URL to return HTTP 200. $2 caps the number of
# attempts (5s apart) so a stuck cert/DNS never hangs the script forever;
# it returns non-zero on timeout instead of looping indefinitely.
wait_for_url() {
  url=$1
  max_attempts=${2:-60}
  attempt=0
  echo "Waiting for $url to return HTTP 200..."

  while [ "$attempt" -lt "$max_attempts" ]; do
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "$url")
    if [ "$status_code" -eq 200 ]; then
      echo "URL ${GREEN}$url${NC} is now accessible (HTTP 200)."
      return 0
    fi

    echo "URL $url is not accessible yet (HTTP $status_code). Retrying in 5 seconds..."
    attempt=$((attempt + 1))
    sleep 5
  done

  echo "URL $url did not become accessible after $((max_attempts * 5)) seconds."
  return 1
}

# Make a request to the api to generate the certificate
curl -s -o /dev/null "https://api.${DOMAIN}"

# Wait for the Stormkit dashboard URL to return HTTP 200
wait_for_url "https://stormkit.${DOMAIN}" ||
  printf "${GRAY}The dashboard is not reachable yet; continuing anyway.${NC}\n"

echo ""

if [ "$DOCKER_MODE" = "Compose" ]; then
  printf "Run ${BLUE}docker compose logs -f${NC} to check your logs"
  echo ""
fi;

print_mcp_config() {
  printf "${GRAY}Admin owner: ${NC}%s\n" "$ADMIN_EMAIL"
  echo ""
  printf "Add this to your MCP client configuration:\n"
  echo ""
  printf "${BLUE}"
  cat <<EOF
{
  "mcpServers": {
    "stormkit": {
      "type": "http",
      "url": "https://api.$DOMAIN/v1/mcp",
      "headers": {
        "Authorization": "Bearer $AGENT_API_KEY"
      }
    }
  }
}
EOF
  printf "${NC}"
  echo ""
  printf "${GRAY}This key has full owner access. It is stored hashed on the server —${NC}\n"
  printf "${GRAY}save it now; it is not recoverable. To rotate it, delete the 'agent'${NC}\n"
  printf "${GRAY}key under User Settings → API Keys and create a new one.${NC}\n"
  echo ""
}

if [ "$AGENT_MODE" = "1" ]; then
  # The server only provisions the admin + key on a *fresh* instance;
  # bootstrap is a no-op when an admin already exists (a re-run, or an
  # instance first set up via OAuth/dashboard). In that case the key we
  # generated was never stored, so confirm it actually authenticates before
  # presenting it as working.
  #
  # A 401/403 is a definitive "not provisioned". Any 2xx means the key
  # authenticated. Anything else (000 from an unreachable host or an api.
  # cert that is not issued yet, a 5xx during boot, ...) is inconclusive:
  # the key is almost certainly valid on a fresh install, but we could not
  # confirm it here, so we say so instead of claiming success.
  auth_status=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "https://api.${DOMAIN}/v1/mcp" \
    -H "Authorization: Bearer $AGENT_API_KEY")

  echo ""

  if [ "$auth_status" = "401" ] || [ "$auth_status" = "403" ]; then
    printf "${GRAY}This instance already has an admin, so no new agent key was provisioned.${NC}\n"
    printf "${GRAY}Manage API keys from the dashboard under User Settings → API Keys.${NC}\n"
    echo ""
  elif [ "$auth_status" -ge 200 ] 2>/dev/null && [ "$auth_status" -lt 300 ]; then
    printf "${GREEN}Your instance is agent-ready.${NC}\n"
    print_mcp_config
  else
    printf "${GRAY}Your agent key was generated but could not be verified yet${NC}"
    printf "${GRAY} (api.$DOMAIN returned '$auth_status').${NC}\n"
    printf "${GRAY}This is expected while the certificate is still being issued.${NC}\n"
    print_mcp_config
  fi
fi;
