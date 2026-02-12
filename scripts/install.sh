#!/usr/bin/env bash
set -euo pipefail

REPO="kiry163/filehub"
TEMPLATE_BASE="https://raw.githubusercontent.com/${REPO}/main/templates"
GH_API="https://api.github.com/repos/${REPO}"
GH_RELEASE_BASE="https://github.com/${REPO}/releases/download"
IMAGE="kirydocker/filehub"

DEFAULT_DIR="${HOME}/.filehub"
DEFAULT_PORT="8080"
DEFAULT_MINIO_PORT="9000"
DEFAULT_MINIO_CONSOLE_PORT="9001"

usage() {
  cat <<'EOF'
FileHub one-click installer (Docker + CLI)

Usage:
  curl -fsSL https://raw.githubusercontent.com/kiry163/filehub/main/scripts/install.sh | bash

Options:
  --dir <path>              Install directory (default: ~/.filehub)
  --port <port>             FileHub port on host (default: 8080)
  --minio-port <port>       MinIO port on host (default: 9000)
  --minio-console-port <p>  MinIO console port on host (default: 9001)
  --version <tag>           Release tag to install (default: latest)
  --help                    Show this help

Notes:
  - First run: generates configuration and starts services
  - Subsequent runs: keeps existing config and data, only ensures services are running
  - To reinstall with fresh config: rm -rf ~/.filehub && run this script again
EOF
}

log() { printf '%s\n' "$*"; }
die() { printf 'Error: %s\n' "$*" >&2; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

http_get() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    wget -qO- "$1"
    return
  fi
  die "need curl or wget"
}

rand_token() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 16
    return
  fi
  tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32
}

compose_cmd() {
  if docker compose version >/dev/null 2>&1; then
    echo "docker compose"
    return
  fi
  if command -v docker-compose >/dev/null 2>&1; then
    echo "docker-compose"
    return
  fi
  die "docker compose not found (need docker compose plugin or docker-compose)"
}

latest_version() {
  local json
  json="$(http_get "${GH_API}/releases/latest")" || return 1
  printf '%s' "$json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}

cli_config_path() {
  local dir
  dir="${XDG_CONFIG_HOME:-${HOME}/.config}"
  echo "${dir}/filehub-cli/config.yaml"
}

install_cli() {
  local version="$1"
  local dest_dir="$2"

  mkdir -p "$dest_dir"
  local url="${GH_RELEASE_BASE}/${version}/filehub-cli-linux-amd64"
  log "Downloading filehub-cli: ${url}"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "${dest_dir}/filehub-cli"
  else
    wget -qO "${dest_dir}/filehub-cli" "$url"
  fi
  chmod +x "${dest_dir}/filehub-cli"
}

fetch_public_ip() {
  local ip=""
  # Try multiple IP services
  if command -v curl >/dev/null 2>&1; then
    ip=$(curl -fsSL --connect-timeout 5 'https://api.ip.sb/jsonip' 2>/dev/null | sed -n 's/.*"ip":[[:space:]]*"\([^"]*\)".*/\1/p')
  fi
  if [[ -z "$ip" ]] && command -v wget >/dev/null 2>&1; then
    ip=$(wget -qO- --timeout=5 'https://api.ip.sb/jsonip' 2>/dev/null | sed -n 's/.*"ip":[[:space:]]*"\([^"]*\)".*/\1/p')
  fi
  printf '%s' "$ip"
}

init_cli_config() {
  local cli_bin="$1"
  local port="$2"
  local local_key="$3"

  "$cli_bin" config init \
    --endpoint "http://localhost:${port}" \
    --local-key "${local_key}" >/dev/null 2>/dev/null || true
}

main() {
  local install_dir="$DEFAULT_DIR"
  local port="$DEFAULT_PORT"
  local minio_port="$DEFAULT_MINIO_PORT"
  local minio_console_port="$DEFAULT_MINIO_CONSOLE_PORT"
  local version=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dir)
        install_dir="$2"; shift 2 ;;
      --port)
        port="$2"; shift 2 ;;
      --minio-port)
        minio_port="$2"; shift 2 ;;
      --minio-console-port)
        minio_console_port="$2"; shift 2 ;;
      --version)
        version="$2"; shift 2 ;;
      --help|-h)
        usage; exit 0 ;;
      *)
        die "unknown argument: $1" ;;
    esac
  done

  need_cmd docker
  local compose
  compose="$(compose_cmd)"

  local image_tag=""
  local release_tag=""
  if [[ -n "$version" ]]; then
    image_tag="$version"
    release_tag="$version"
  else
    log "Trying to pull latest image..."
    if docker pull "${IMAGE}:latest" >/dev/null 2>&1; then
      image_tag="latest"
    else
      log "Latest image not available, falling back to GitHub releases tag..."
    fi
    release_tag="$(latest_version)"
    [[ -n "$release_tag" ]] || die "failed to resolve latest release tag; try --version vX.Y.Z"
    if [[ -z "$image_tag" ]]; then
      image_tag="$release_tag"
    fi
  fi

  mkdir -p "$install_dir/data/minio"
  cd "$install_dir"

  local config_file="${install_dir}/config.yaml"
  local compose_file="${install_dir}/docker-compose.yml"
  local local_key=""

  # Check if this is first-time installation
  if [[ -f "$config_file" ]]; then
    log "Existing configuration found at ${config_file}"
    log "Keeping existing configuration and data unchanged"
    log "To reinstall with fresh config, delete ${install_dir} and run again"
    log ""
    
    # Extract local_key from existing config
    local_key="$(grep "local_key:" "$config_file" | sed 's/.*local_key:[[:space:]]*"\([^"]*\)".*/\1/')"
  else
    log "First time installation, generating configuration..."
    log ""

    # Generate random values
    local jwt_secret admin_password minio_secret public_endpoint
    jwt_secret="$(rand_token)$(rand_token)"
    admin_password="$(rand_token)"
    local_key="$(rand_token)"
    minio_secret="$(rand_token)"

    # Try to detect public IP
    log "Detecting public IP..."
    local public_ip
    public_ip="$(fetch_public_ip)"
    if [[ -n "$public_ip" ]]; then
      public_endpoint="http://${public_ip}:${port}"
      log "Public endpoint detected: ${public_endpoint}"
    else
      public_endpoint=""
      log "Could not detect public IP, public_endpoint will be empty"
    fi

    # Download templates
    log "Downloading configuration templates..."
    local config_template compose_template
    config_template="$(mktemp)"
    compose_template="$(mktemp)"
    
    http_get "${TEMPLATE_BASE}/config.yaml.template" > "$config_template"
    http_get "${TEMPLATE_BASE}/docker-compose.yaml.template" > "$compose_template"

    # Render config template
    sed -e "s/{{JWT_SECRET}}/${jwt_secret}/g" \
        -e "s/{{ADMIN_PASSWORD}}/${admin_password}/g" \
        -e "s/{{LOCAL_KEY}}/${local_key}/g" \
        -e "s/{{MINIO_SECRET_KEY}}/${minio_secret}/g" \
        -e "s|{{PUBLIC_ENDPOINT}}|${public_endpoint}|g" \
        "$config_template" > "$config_file"

    # Render compose template
    sed -e "s/{{VERSION}}/${image_tag}/g" \
        -e "s/{{SERVER_PORT}}/${port}/g" \
        -e "s/{{MINIO_PORT}}/${minio_port}/g" \
        -e "s/{{MINIO_CONSOLE_PORT}}/${minio_console_port}/g" \
        -e "s/{{MINIO_SECRET_KEY}}/${minio_secret}/g" \
        "$compose_template" > "$compose_file"

    # Cleanup temp files
    rm -f "$config_template" "$compose_template"

    log "Configuration generated successfully!"
    log ""
    log "Credentials (save these now):"
    log "- admin username: admin"
    log "- admin password: ${admin_password}"
    log "- local_key:      ${local_key}"
    log ""
  fi

  # Ensure docker-compose.yml exists (in case user deleted it but kept config)
  if [[ ! -f "$compose_file" ]]; then
    log "docker-compose.yml not found, regenerating from template..."
    local compose_template
    compose_template="$(mktemp)"
    http_get "${TEMPLATE_BASE}/docker-compose.yaml.template" > "$compose_template"
    
    # Extract minio_secret from existing config
    local minio_secret
    minio_secret="$(grep "secret_key:" "$config_file" | sed 's/.*secret_key:[[:space:]]*"\([^"]*\)".*/\1/')"
    
    sed -e "s/{{VERSION}}/${image_tag}/g" \
        -e "s/{{SERVER_PORT}}/${port}/g" \
        -e "s/{{MINIO_PORT}}/${minio_port}/g" \
        -e "s/{{MINIO_CONSOLE_PORT}}/${minio_console_port}/g" \
        -e "s/{{MINIO_SECRET_KEY}}/${minio_secret}/g" \
        "$compose_template" > "$compose_file"
    
    rm -f "$compose_template"
  fi

  # Pull and start services
  log "Pulling images..."
  docker pull "${IMAGE}:${image_tag}" >/dev/null 2>&1 || docker pull "${IMAGE}:${image_tag}"
  docker pull "minio/minio:latest" >/dev/null 2>&1 || docker pull "minio/minio:latest"

  log "Starting services..."
  ${compose} up -d

  # Install CLI
  local cli_dir="/usr/local/bin"
  if [[ ! -w "/usr/local/bin" ]]; then
    cli_dir="${HOME}/.local/bin"
    log "Warning: /usr/local/bin not writable, installing CLI to ${cli_dir}"
  fi

  install_cli "$release_tag" "$cli_dir"

  local cli_config
  cli_config="$(cli_config_path)"
  if [[ ! -f "$cli_config" && -n "$local_key" ]]; then
    init_cli_config "${cli_dir}/filehub-cli" "$port" "$local_key"
  else
    log "CLI config exists, keeping it unchanged"
  fi

  log ""
  log "FileHub ${image_tag} is ready!"
  log "- Web UI:       http://localhost:${port}"
  log "- API:          http://localhost:${port}/api/v1"
  log "- MinIO UI:     http://localhost:${minio_console_port}"
  log ""
  log "CLI installed: ${cli_dir}/filehub-cli"
  log ""
  log "Manage services: cd ${install_dir} && ${compose} up -d | ${compose} down"
  log "View logs:       cd ${install_dir} && ${compose} logs -f"
}

main "$@"
