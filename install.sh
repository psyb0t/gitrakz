#!/bin/bash
set -euo pipefail

# gitrakz installer. Run with:
#   curl -fsSL https://raw.githubusercontent.com/psyb0t/gitrakz/main/install.sh | sudo bash
#
# It pins the local stack to the latest RELEASE tag (never :latest on your box),
# drops docker-compose.yml + an owner-only .env into ~/.gitrakz, and installs a
# `gitrakz` command that wraps the usual Docker commands.

readonly INSTALL_LOG_FILE="/tmp/gitrakz-install.log"
readonly WRAPPER_PATH="/usr/local/bin/gitrakz"
readonly WRAPPER_MARKER="gitrakz-managed-command"
readonly CONFIG_DIRECTORY_NAME=".gitrakz"
readonly IMAGE_REPO="psyb0t/gitrakz"
readonly RAW_BASE="https://raw.githubusercontent.com/psyb0t/gitrakz"
readonly GITHUB_API_LATEST="https://api.github.com/repos/psyb0t/gitrakz/releases/latest"

WRAPPER_TEMPORARY_FILE=""

log() {
	local level="$1"
	shift

	printf '{"time":"%s","level":"%s","file":"%s","line":%d,"func":"%s","msg":"%s"}\n' \
		"$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
		"$level" \
		"${BASH_SOURCE[1]##*/}" \
		"${BASH_LINENO[0]}" \
		"${FUNCNAME[1]:-main}" \
		"$*" >&2
}

trap 'log ERROR "command failed exit=$?"' ERR
trap 'rm -f "$WRAPPER_TEMPORARY_FILE"' EXIT
exec > >(tee -a "$INSTALL_LOG_FILE") 2>&1

fail() {
	log ERROR "$*"
	exit 1
}

require_root() {
	if ((EUID != 0)); then
		fail "run this installer with sudo"
	fi

	if [[ -z "${SUDO_USER:-}" || "$SUDO_USER" == "root" ]]; then
		fail "run this through sudo from the account that will use gitrakz"
	fi
}

resolve_target_user() {
	local account_record

	account_record="$(getent passwd "$SUDO_USER")"
	[[ -n "$account_record" ]] || fail "could not resolve sudo user $SUDO_USER"

	TARGET_HOME="$(cut -d: -f6 <<<"$account_record")"
	TARGET_UID="$(id -u "$SUDO_USER")"
	TARGET_GID="$(id -g "$SUDO_USER")"
	TARGET_CONFIG_DIR="$TARGET_HOME/$CONFIG_DIRECTORY_NAME"
}

run_as_target_user() {
	sudo -H -u "$SUDO_USER" "$@"
}

# ensure_gh installs the GitHub CLI on the host if missing; the wrapper needs it
# to read a token with `gh auth token`.
ensure_gh() {
	if command -v gh >/dev/null 2>&1; then
		log INFO "gh is already installed"

		return
	fi

	log INFO "installing the GitHub CLI (gh)"

	if command -v apk >/dev/null 2>&1; then
		apk add --no-cache github-cli
	elif command -v dnf >/dev/null 2>&1; then
		dnf install -y gh
	elif command -v pacman >/dev/null 2>&1; then
		pacman -Sy --noconfirm github-cli
	elif command -v zypper >/dev/null 2>&1; then
		zypper --non-interactive install gh
	elif command -v apt-get >/dev/null 2>&1; then
		install_gh_apt
	else
		fail "no supported package manager found; install gh from https://cli.github.com and re-run"
	fi

	command -v gh >/dev/null 2>&1 ||
		fail "gh install did not produce a gh command; see https://cli.github.com"
}

# install_gh_apt wires GitHub's official apt repository, because gh is not in
# every Debian/Ubuntu default repo.
install_gh_apt() {
	local keyring="/etc/apt/keyrings/githubcli-archive-keyring.gpg"

	export DEBIAN_FRONTEND=noninteractive
	apt-get update
	apt-get install -y curl ca-certificates gpg
	install -m 0755 -d /etc/apt/keyrings
	curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg |
		gpg --dearmor -o "$keyring"
	chmod go+r "$keyring"
	printf 'deb [arch=%s signed-by=%s] https://cli.github.com/packages stable main\n' \
		"$(dpkg --print-architecture)" "$keyring" \
		>/etc/apt/sources.list.d/github-cli.list
	apt-get update
	apt-get install -y gh
}

resolve_latest_tag() {
	local tag

	tag="$(curl -fsSL "$GITHUB_API_LATEST" |
		sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
	[[ -n "$tag" ]] || fail "could not resolve the latest gitrakz release tag"

	printf '%s\n' "$tag"
}

env_pinned_image() {
	local env_file="$1"

	[[ -f "$env_file" ]] || return 0
	sed -n 's/^GITRAKZ_IMAGE=//p' "$env_file" | head -n1
}

set_env_var() {
	local env_file="$1" key="$2" value="$3"

	if grep -q "^${key}=" "$env_file"; then
		sed -i "s|^${key}=.*|${key}=${value}|" "$env_file"
	else
		printf '%s=%s\n' "$key" "$value" >>"$env_file"
	fi
}

write_config() {
	local tag="$1" image="$IMAGE_REPO:$1"

	install -d -m 0700 -o "$TARGET_UID" -g "$TARGET_GID" "$TARGET_CONFIG_DIR"

	# The compose file is framework — always refresh it to the pinned tag.
	curl -fsSL "$RAW_BASE/$tag/docker-compose.yml" \
		-o "$TARGET_CONFIG_DIR/docker-compose.yml"

	# The .env is the user's — seed it once, then only update the image pin so a
	# re-run (upgrade) never clobbers their AUTH_TOKEN / GH_USER / etc.
	if [[ ! -f "$TARGET_CONFIG_DIR/.env" ]]; then
		curl -fsSL "$RAW_BASE/$tag/.env.example" -o "$TARGET_CONFIG_DIR/.env"
	fi
	set_env_var "$TARGET_CONFIG_DIR/.env" GITRAKZ_IMAGE "$image"

	chown -R "$TARGET_UID:$TARGET_GID" "$TARGET_CONFIG_DIR"
	chmod 0600 "$TARGET_CONFIG_DIR/.env"
}

write_wrapper() {
	if [[ -e "$WRAPPER_PATH" ]] && ! grep -Fq "$WRAPPER_MARKER" "$WRAPPER_PATH"; then
		fail "$WRAPPER_PATH already exists and is not managed by this installer"
	fi

	WRAPPER_TEMPORARY_FILE="$(mktemp)"

	cat >"$WRAPPER_TEMPORARY_FILE" <<'WRAPPER'
#!/bin/bash
# gitrakz-managed-command
set -euo pipefail

readonly CONFIG_DIRECTORY_NAME=".gitrakz"
readonly IMAGE_REPO="psyb0t/gitrakz"
readonly ROLLING_IMAGE="psyb0t/gitrakz:latest"
readonly RAW_BASE="https://raw.githubusercontent.com/psyb0t/gitrakz"
readonly INSTALLER_URL="https://raw.githubusercontent.com/psyb0t/gitrakz/main/install.sh"
readonly GITHUB_API_LATEST="https://api.github.com/repos/psyb0t/gitrakz/releases/latest"
readonly COMMAND_LOG_FILE="${TMPDIR:-/tmp}/gitrakz-command.log"

log() {
	local level="$1"
	shift

	printf '{"time":"%s","level":"%s","file":"%s","line":%d,"func":"%s","msg":"%s"}\n' \
		"$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
		"$level" \
		"${BASH_SOURCE[1]##*/}" \
		"${BASH_LINENO[0]}" \
		"${FUNCNAME[1]:-main}" \
		"$*" >&2
}

trap 'log ERROR "command failed exit=$?"' ERR
exec > >(tee -a "$COMMAND_LOG_FILE") 2>&1

fail() {
	log ERROR "$*"
	exit 1
}

usage() {
	cat <<'EOF'
Usage: gitrakz <command> [--rolling]

Commands:
  setup      Create ~/.gitrakz (compose + .env) without replacing your config
  start      Inject a GitHub token from `gh auth token`, pull, and start it
  stop       Stop the gitrakz stack
  status     Show the container state
  logs       Follow logs; pass any extra `docker compose logs` arguments
  upgrade    Re-pin to the latest release, pull it, and drop the previous image
  uninstall  Stop the stack, remove the command, and offer to delete your data
  help       Show this help

  --rolling  On start/upgrade: use the moving :latest image for that run only,
             instead of the pinned release tag. Handy for testing main.
EOF
}

config_directory() {
	local config_dir="${GITRAKZ_HOME:-$HOME/$CONFIG_DIRECTORY_NAME}"

	[[ "$config_dir" = /* ]] || fail "GITRAKZ_HOME must be an absolute path"
	printf '%s\n' "$config_dir"
}

require_docker() {
	command -v docker >/dev/null 2>&1 || fail "docker is not installed"
	docker info >/dev/null 2>&1 ||
		fail "cannot talk to Docker — is the daemon running and are you in the docker group?"
}

compose() {
	local config_dir="$1"
	shift

	docker compose \
		--project-directory "$config_dir" \
		--env-file "$config_dir/.env" \
		-f "$config_dir/docker-compose.yml" "$@"
}

ensure_config() {
	local config_dir="$1"

	[[ -f "$config_dir/docker-compose.yml" && -f "$config_dir/.env" ]] ||
		fail "no config at $config_dir — run: gitrakz setup"
}

env_get() {
	sed -n "s/^$2=//p" "$1/.env" | head -n1
}

env_set() {
	local config_dir="$1" key="$2" value="$3"

	if grep -q "^${key}=" "$config_dir/.env"; then
		sed -i "s|^${key}=.*|${key}=${value}|" "$config_dir/.env"
	else
		printf '%s=%s\n' "$key" "$value" >>"$config_dir/.env"
	fi
}

gh_token() {
	command -v gh >/dev/null 2>&1 || fail "gh is not installed — see https://cli.github.com"

	local token
	token="$(gh auth token 2>/dev/null || true)"
	[[ -n "$token" ]] || fail "gh is not authenticated — run: gh auth login"

	printf '%s\n' "$token"
}

resolve_latest_tag() {
	local tag
	tag="$(curl -fsSL "$GITHUB_API_LATEST" |
		sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
	[[ -n "$tag" ]] || fail "could not resolve the latest gitrakz release tag"

	printf '%s\n' "$tag"
}

setup() {
	local config_dir tag
	config_dir="$(config_directory)"
	mkdir -p "$config_dir"
	chmod 700 "$config_dir"
	tag="$(resolve_latest_tag)"

	curl -fsSL "$RAW_BASE/$tag/docker-compose.yml" \
		-o "$config_dir/docker-compose.yml"
	if [[ ! -f "$config_dir/.env" ]]; then
		curl -fsSL "$RAW_BASE/$tag/.env.example" -o "$config_dir/.env"
		chmod 600 "$config_dir/.env"
		env_set "$config_dir" GITRAKZ_IMAGE "$IMAGE_REPO:$tag"
	fi

	log INFO "config ready at $config_dir"
}

start() {
	require_docker
	local config_dir
	config_dir="$(config_directory)"
	ensure_config "$config_dir"

	export GH_TOKEN
	GH_TOKEN="$(gh_token)"

	if [[ "${GITRAKZ_ROLLING:-}" == "1" ]]; then
		log WARN "running the rolling :latest image for this run"
		export GITRAKZ_IMAGE="$ROLLING_IMAGE"
	fi

	compose "$config_dir" up --detach --pull always
	compose "$config_dir" ps
}

stop() {
	require_docker
	local config_dir
	config_dir="$(config_directory)"
	ensure_config "$config_dir"
	compose "$config_dir" down
}

status() {
	require_docker
	local config_dir
	config_dir="$(config_directory)"
	ensure_config "$config_dir"
	compose "$config_dir" ps
}

show_logs() {
	require_docker
	local config_dir
	config_dir="$(config_directory)"
	ensure_config "$config_dir"
	compose "$config_dir" logs "$@"
}

upgrade() {
	require_docker
	local config_dir
	config_dir="$(config_directory)"
	ensure_config "$config_dir"

	if [[ "${GITRAKZ_ROLLING:-}" == "1" ]]; then
		log WARN "pulling the rolling :latest image (pin unchanged)"
		export GITRAKZ_IMAGE="$ROLLING_IMAGE"
		export GH_TOKEN
		GH_TOKEN="$(gh_token)"
		docker pull "$ROLLING_IMAGE"
		compose "$config_dir" up --detach
		compose "$config_dir" ps

		return
	fi

	local old_image new_tag new_image
	old_image="$(env_get "$config_dir" GITRAKZ_IMAGE)"
	new_tag="$(resolve_latest_tag)"
	new_image="$IMAGE_REPO:$new_tag"

	if [[ "$old_image" == "$new_image" ]]; then
		log INFO "already on the latest release $new_tag; re-pulling"
	fi

	docker pull "$new_image"
	curl -fsSL "$RAW_BASE/$new_tag/docker-compose.yml" \
		-o "$config_dir/docker-compose.yml"
	env_set "$config_dir" GITRAKZ_IMAGE "$new_image"

	export GH_TOKEN
	GH_TOKEN="$(gh_token)"
	compose "$config_dir" up --detach
	compose "$config_dir" ps

	# Refresh the wrapper itself from the new installer (self-update).
	log INFO "refreshing the installer + wrapper"
	curl -fsSL "$INSTALLER_URL" | sudo bash

	# Reclaim space: drop the previous image once the new one is running.
	if [[ -n "$old_image" && "$old_image" != "$new_image" ]]; then
		if docker image rm "$old_image" >/dev/null 2>&1; then
			log INFO "removed the previous image $old_image"
		else
			log WARN "kept $old_image (still in use); prune it later for the space"
		fi
	fi
}

uninstall() {
	require_docker
	local config_dir answer delete_data=0
	config_dir="$(config_directory)"

	read -r -p "Also delete your data ($config_dir and the gitrakz data volume)? [y/N] " answer
	case "$answer" in
	y | Y | yes | YES) delete_data=1 ;;
	esac

	if [[ -f "$config_dir/docker-compose.yml" ]]; then
		log INFO "stopping the stack"
		if ((delete_data)); then
			compose "$config_dir" down --volumes --remove-orphans || true
		else
			compose "$config_dir" down --remove-orphans || true
		fi
	fi

	log INFO "removing the gitrakz command (needs sudo)"
	sudo rm -f /usr/local/bin/gitrakz

	if ((delete_data)); then
		rm -rf -- "$config_dir"
		log INFO "deleted $config_dir and the data volume"
	else
		log INFO "kept your data at $config_dir"
	fi

	log INFO "gitrakz uninstalled"
}

main() {
	local command="${1:-help}"
	shift || true

	# A single global flag: --rolling forces the moving :latest image.
	local -a rest=()
	local arg
	for arg in "$@"; do
		case "$arg" in
		--rolling) export GITRAKZ_ROLLING=1 ;;
		*) rest+=("$arg") ;;
		esac
	done

	case "$command" in
	setup) setup ;;
	start) start ;;
	stop) stop ;;
	status) status ;;
	logs) show_logs "${rest[@]}" ;;
	upgrade) upgrade ;;
	uninstall) uninstall ;;
	help | --help | -h) usage ;;
	*) fail "unknown command: $command (try: gitrakz help)" ;;
	esac
}

main "$@"
WRAPPER

	install -m 0755 "$WRAPPER_TEMPORARY_FILE" "$WRAPPER_PATH"
	log INFO "installed the gitrakz command at $WRAPPER_PATH"
}

main() {
	require_root
	resolve_target_user

	log DEBUG "checking Docker access for the target user"
	run_as_target_user docker info >/dev/null

	ensure_gh

	local previous_image new_tag new_image
	previous_image="$(env_pinned_image "$TARGET_CONFIG_DIR/.env")"
	new_tag="$(resolve_latest_tag)"
	new_image="$IMAGE_REPO:$new_tag"

	log INFO "pinning to the latest release $new_tag"
	write_config "$new_tag"
	write_wrapper

	log INFO "pulling $new_image"
	run_as_target_user docker pull "$new_image"

	if [[ -n "$previous_image" && "$previous_image" != "$new_image" ]]; then
		if run_as_target_user docker image rm "$previous_image" >/dev/null 2>&1; then
			log INFO "removed the previous image $previous_image"
		else
			log WARN "kept the previous image $previous_image (still in use)"
		fi
	fi

	printf '\ngitrakz is installed (pinned to %s).\n\n' "$new_tag"
	printf '  Config:  %s/.env\n' "$TARGET_CONFIG_DIR"
	printf '  Command: %s\n\n' "$WRAPPER_PATH"
	printf 'Edit the .env to change the published port, expose it beyond localhost,\n'
	printf 'or enable LLM features (set the GITRAKZ_ELELEM_* vars) — all optional.\n\n'
	printf 'Authenticate GitHub once (the wrapper reads the token from it):\n\n'
	printf '  gh auth login\n\n'
	printf 'Then start it (tracks your own activity by default):\n\n'
	printf '  gitrakz start\n\n'
	printf 'Then open the published port (default http://127.0.0.1:8080) and watch\n'
	printf 'gitrakz status / gitrakz logs -f.\n\n'
}

main "$@"
