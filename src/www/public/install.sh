#!/bin/sh

# curl -sSL https://www.stormkit.io/install.sh | sh

# Function to check if a command exists
command_exists() {
  command -v "$@" >/dev/null 2>&1
}

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

setup_base_env_variables() {
  # Download the example .env file
  curl -o ".env" "https://raw.githubusercontent.com/stormkit-io/bin/main/.env.example" --silent

  update_env_var_in_env_file POSTGRES_PASSWORD $(openssl rand -base64 12 | tr -dc 'a-zA-Z0-9' | head -c24)
  update_env_var_in_env_file STORMKIT_APP_SECRET $(openssl rand -base64 48 | tr -dc 'a-zA-Z0-9' | head -c32)
}

DOMAIN=""

# Setup the environment variable for the Hosting Service.
setup_domain() {
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

# For now keep this file here - we need to improve the docker swarm setup
# Download the docker-compose.yaml file
curl -o "docker-compose.yaml" "https://raw.githubusercontent.com/stormkit-io/bin/main/docker-compose.yaml" --silent

setup_base_env_variables
setup_domain

docker compose up -d

echo ""
printf "${GREEN}Congratulations, Stormkit is installed on your computer!\n${NC}"

# Function to wait for the URL to return HTTP 200
wait_for_url() {
  url=$1
  echo "Waiting for $url to return HTTP 200..."

  while true; do
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "$url")
    if [ "$status_code" -eq 200 ]; then
      echo "URL ${GREEN}$url${NC} is now accessible (HTTP 200)."
      break
    else
      echo "URL $url is not accessible yet (HTTP $status_code). Retrying in 5 seconds..."
      sleep 5
    fi
  done
}

# Make a request to the api to generate the certificate
curl -s -o /dev/null "https://api.${DOMAIN}"

# Wait for the Stormkit dashboard URL to return HTTP 200
wait_for_url "https://stormkit.${DOMAIN}"

echo ""

if [ "$DOCKER_MODE" = "Compose" ]; then
  printf "Run ${BLUE}docker compose logs -f${NC} to check your logs"
  echo ""
fi;
