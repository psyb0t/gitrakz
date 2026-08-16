#!/bin/bash
set -euo pipefail

# gitrakz installer. Two modes:
#
#   Per-user (no root) — installs into your home, for the current user only:
#     curl -fsSL https://raw.githubusercontent.com/psyb0t/gitrakz/main/install.sh | bash
#     command -> ~/.local/bin/gitrakz, config -> ~/.config/gitrakz
#
#   System-wide (root) — one shared stack any docker-group user can drive:
#     curl -fsSL https://raw.githubusercontent.com/psyb0t/gitrakz/main/install.sh | sudo bash
#     command -> /usr/local/bin/gitrakz, config -> /etc/gitrakz
#
# The mode is chosen from who runs it (root -> system, otherwise per-user); pass
# --system or --user to force it. It pins the stack to the latest RELEASE tag
# (never :latest on your box).

readonly INSTALL_LOG_FILE="/tmp/gitrakz-install.log"
readonly SYSTEM_INSTALL_PATH="/usr/local/bin/gitrakz"
readonly SYSTEM_CONFIG_DIR="/etc/gitrakz"
readonly WRAPPER_MARKER="gitrakz-managed-command"
readonly CONFIG_DIRECTORY_NAME="gitrakz"
readonly DATA_DIRECTORY_NAME="data"
readonly CONTAINER_DATA_OWNER="1000:1000"
# shellcheck disable=SC2016  # deliberately literal: $HOME/$PATH must land in the rc file unexpanded
readonly USER_PATH_SNIPPET='export PATH="$HOME/.local/bin:$PATH"'
readonly IMAGE_REPO="psyb0t/gitrakz"
readonly RAW_BASE="https://raw.githubusercontent.com/psyb0t/gitrakz"
readonly GITHUB_API_LATEST="https://api.github.com/repos/psyb0t/gitrakz/releases/latest"

MODE=""
INSTALL_PATH=""
TARGET_CONFIG_DIR=""
WRAPPER_TEMPORARY_FILE=""

# This is a user-facing installer, so the output is deliberately plain prose
# (per the bash logging rule's carve-out for CLI output the user asked for),
# not JSON. Everything is still tee'd to INSTALL_LOG_FILE for a debug trail.
say() { printf '==> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }

fail() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

trap 'printf "error: install failed — see %s\n" "$INSTALL_LOG_FILE" >&2' ERR
trap 'rm -f "$WRAPPER_TEMPORARY_FILE"' EXIT
exec > >(tee -a "$INSTALL_LOG_FILE") 2>&1

# resolve_mode fixes MODE (from --system/--user, else EUID) and the paths it
# implies, and rejects the nonsensical combinations up front.
resolve_mode() {
	if [[ -z "$MODE" ]]; then
		if ((EUID == 0)); then MODE="system"; else MODE="user"; fi
	fi

	case "$MODE" in
	system)
		((EUID == 0)) ||
			fail "the system-wide install needs root — re-run with sudo, or use --user for a per-user install"
		INSTALL_PATH="$SYSTEM_INSTALL_PATH"
		TARGET_CONFIG_DIR="$SYSTEM_CONFIG_DIR"
		;;
	user)
		((EUID != 0)) ||
			fail "the per-user install must not run as root — run it as your normal account, or use --system"
		INSTALL_PATH="$HOME/.local/bin/gitrakz"
		TARGET_CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/$CONFIG_DIRECTORY_NAME"
		;;
	*)
		fail "unknown mode: $MODE"
		;;
	esac
}

# prepare_config_dir creates the config directory with mode-appropriate
# ownership. System config is root-owned but group-readable by the docker group
# so any docker-group operator can drive the shared stack (docker-group access
# is already root-equivalent, so this exposes nothing new). Per-user config is
# private (0700) to the installing user.
prepare_config_dir() {
	if [[ "$MODE" == "system" ]]; then
		install -d -m 0750 "$TARGET_CONFIG_DIR"
		if getent group docker >/dev/null 2>&1; then
			chgrp docker "$TARGET_CONFIG_DIR"
		fi

		return
	fi

	install -d -m 0700 "$TARGET_CONFIG_DIR"
}

# prepare_data_dir makes persistence visible beside docker-compose.yml rather
# than burying it in a Docker-managed volume. The image's appuser is fixed to
# 1000:1000; per-user stacks map that identity to the installing account when
# the wrapper starts Compose.
prepare_data_dir() {
	local data_dir="$TARGET_CONFIG_DIR/$DATA_DIRECTORY_NAME"

	if [[ "$MODE" == "system" ]]; then
		install -d -m 0750 -o 1000 -g 1000 "$data_dir"

		return
	fi

	install -d -m 0700 "$data_dir"
}

data_owner() {
	if [[ "$MODE" == "system" ]]; then
		printf '%s\n' "$CONTAINER_DATA_OWNER"

		return
	fi

	printf '%s:%s\n' "$(id -u)" "$(id -g)"
}

# legacy_data_volume returns the old Compose-managed volume only while the
# installed Compose file still references it. New installs use ./data instead.
legacy_data_volume() {
	local config_dir="$1"

	[[ -f "$config_dir/docker-compose.yml" && -f "$config_dir/.env" ]] || return 0
	grep -Fq 'gitrakz-data:/data' "$config_dir/docker-compose.yml" || return 0

	GH_TOKEN=unused GITRAKZ_CONTAINER_USER="$CONTAINER_DATA_OWNER" docker compose \
		--project-directory "$config_dir" \
		--env-file "$config_dir/.env" \
		-f "$config_dir/docker-compose.yml" config 2>/dev/null | awk '
			$0 == "volumes:" { in_volumes = 1; next }
			in_volumes && $0 == "  gitrakz-data:" { target = 1; next }
			target && $1 == "name:" { print $2; exit }
		'
}

# migrate_legacy_data makes the named-volume database visible in the install
# directory before a refreshed Compose file switches to the bind mount. The old
# volume is retained as a harmless rollback copy; this function never removes it.
migrate_legacy_data() {
	local volume_name="$1" image="$2" data_dir="$TARGET_CONFIG_DIR/$DATA_DIRECTORY_NAME"
	local owner

	[[ -n "$volume_name" && ! -e "$data_dir/gitrakz.db" ]] || return 0
	docker volume inspect "$volume_name" >/dev/null 2>&1 || return 0
	owner="$(data_owner)"

	say "moving existing gitrakz data into $data_dir"
	docker run --rm --user 0:0 --entrypoint sh \
		--mount "type=volume,source=$volume_name,target=/from,readonly" \
		--mount "type=bind,source=$data_dir,target=/to" \
		"$image" -ec 'cp -a /from/. /to/ && chown -R "$1" /to' sh "$owner"
}

# apply_config_permissions re-asserts the sharing model after config is written.
# Only meaningful for the system install.
apply_config_permissions() {
	[[ "$MODE" == "system" ]] || return 0

	if getent group docker >/dev/null 2>&1; then
		find "$TARGET_CONFIG_DIR" -path "$TARGET_CONFIG_DIR/$DATA_DIRECTORY_NAME" -prune -o -exec chgrp docker {} + || true
	fi
	# Group may read/traverse but never write.
	find "$TARGET_CONFIG_DIR" -path "$TARGET_CONFIG_DIR/$DATA_DIRECTORY_NAME" -prune -o -exec chmod g-w {} + || true
	find "$TARGET_CONFIG_DIR" -path "$TARGET_CONFIG_DIR/$DATA_DIRECTORY_NAME" -prune -o -type d -exec chmod g+rx {} + || true
	find "$TARGET_CONFIG_DIR" -path "$TARGET_CONFIG_DIR/$DATA_DIRECTORY_NAME" -prune -o -type f -exec chmod g+r {} + || true
}

# warn_user_path tells a per-user installer, in the terminal, exactly how to put
# ~/.local/bin on PATH for both bash and zsh when it is not already there.
warn_user_path() {
	[[ "$MODE" == "user" ]] || return 0

	case ":$PATH:" in
	*":$HOME/.local/bin:"*) return 0 ;;
	esac

	warn "$HOME/.local/bin is not on your PATH — the gitrakz command will not be found yet"
	printf '\nAdd it to your shell, then restart the shell (or source the file):\n\n'
	printf "  bash:  echo '%s' >> ~/.bashrc && source ~/.bashrc\n" "$USER_PATH_SNIPPET"
	printf "  zsh:   echo '%s' >> ~/.zshrc && source ~/.zshrc\n\n" "$USER_PATH_SNIPPET"
}

# ensure_gh makes sure the GitHub CLI is available. Installing it needs a package
# manager and root, so only the system install can install it; a per-user
# install just checks for it and points the user at the download when missing.
ensure_gh() {
	if command -v gh >/dev/null 2>&1; then
		say "gh is already installed"

		return
	fi

	if [[ "$MODE" == "user" ]]; then
		warn "the GitHub CLI (gh) is not installed"
		printf '\nA per-user install cannot install gh (that needs root). Install it from\n'
		printf 'https://cli.github.com, or re-run with sudo for a system-wide install\n'
		printf 'that installs gh for you. gitrakz reads its token from gh auth token.\n\n'

		return
	fi

	say "installing the GitHub CLI (gh)"
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

# write_config drops the pinned compose file + a seeded .env into the config dir.
# The dir already has mode-appropriate ownership from prepare_config_dir; files
# are owned by the running identity (the user, or root for the system install).
write_config() {
	local tag="$1" image="$IMAGE_REPO:$1"

	# The compose file is framework — always refresh it to the pinned tag.
	curl -fsSL "$RAW_BASE/$tag/docker-compose.yml" \
		-o "$TARGET_CONFIG_DIR/docker-compose.yml"

	# The .env is the user's — seed it once, then only update the image pin so a
	# re-run (upgrade) never clobbers their AUTH_TOKEN / GH_USER / etc.
	if [[ ! -f "$TARGET_CONFIG_DIR/.env" ]]; then
		curl -fsSL "$RAW_BASE/$tag/.env.example" -o "$TARGET_CONFIG_DIR/.env"
	fi
	set_env_var "$TARGET_CONFIG_DIR/.env" GITRAKZ_IMAGE "$image"
	chmod 0600 "$TARGET_CONFIG_DIR/.env"
}

write_command() {
	if [[ -e "$INSTALL_PATH" ]] && ! grep -Fq "$WRAPPER_MARKER" "$INSTALL_PATH"; then
		fail "$INSTALL_PATH already exists and is not managed by this installer"
	fi

	install -d "$(dirname "$INSTALL_PATH")"
	WRAPPER_TEMPORARY_FILE="$(mktemp)"

	cat >"$WRAPPER_TEMPORARY_FILE" <<'SCRIPT'
#!/bin/bash
# gitrakz-managed-command
set -euo pipefail

readonly CONFIG_DIRECTORY_NAME="gitrakz"
readonly SYSTEM_CONFIG_DIR="/etc/gitrakz"
readonly IMAGE_REPO="psyb0t/gitrakz"
readonly ROLLING_IMAGE="psyb0t/gitrakz:latest"
readonly RAW_BASE="https://raw.githubusercontent.com/psyb0t/gitrakz"
readonly INSTALLER_URL="https://raw.githubusercontent.com/psyb0t/gitrakz/main/install.sh"
readonly GITHUB_API_LATEST="https://api.github.com/repos/psyb0t/gitrakz/releases/latest"
readonly COMMAND_LOG_FILE="${TMPDIR:-/tmp}/gitrakz-command.log"

# Plain user-facing output; still tee'd to COMMAND_LOG_FILE for a debug trail.
say() { printf '==> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }

fail() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

trap 'printf "error: command failed — see %s\n" "$COMMAND_LOG_FILE" >&2' ERR
exec > >(tee -a "$COMMAND_LOG_FILE") 2>&1

usage() {
    cat <<'EOF'
Usage: gitrakz <command> [--rolling]

Commands:
  setup      Create the config (compose + .env) without replacing your config
  start      Inject a GitHub token from `gh auth token`, pull, and start it
  stop       Stop the gitrakz stack
  status     Show the container state
  logs       Follow logs; pass any extra `docker compose logs` arguments
  upgrade    Re-pin to the latest release, pull it, and drop the previous image
  uninstall  Stop the stack, remove the command, and offer to delete your data
  help       Show this help

  --rolling  On start/upgrade: use the moving :latest image for that run only,
             instead of the pinned release tag. Handy for testing main.

Config location: $GITRAKZ_HOME if set, else /etc/gitrakz for a system-wide
install, else ~/.config/gitrakz for a per-user install.
EOF
}

# config_directory resolves where the stack config lives: an explicit
# GITRAKZ_HOME wins; otherwise a system-wide install (/etc/gitrakz) is preferred
# when present, falling back to the per-user ~/.config/gitrakz.
config_directory() {
    local config_dir
    if [[ -n "${GITRAKZ_HOME:-}" ]]; then
        config_dir="$GITRAKZ_HOME"
    elif [[ -f "$SYSTEM_CONFIG_DIR/docker-compose.yml" ]]; then
        config_dir="$SYSTEM_CONFIG_DIR"
    else
        config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/$CONFIG_DIRECTORY_NAME"
    fi

    [[ "$config_dir" = /* ]] || fail "GITRAKZ_HOME must be an absolute path"
    printf '%s\n' "$config_dir"
}

# root_wrap prints the sudo prefix needed to write to a config dir the current
# user does not own (i.e. the system-wide /etc/gitrakz). Empty for per-user.
root_wrap() {
    local config_dir="$1"
    if [[ -w "$config_dir" ]]; then
        return
    fi
    command -v sudo >/dev/null || fail "writing $config_dir needs root but sudo is not available"
    printf 'sudo'
}

require_docker() {
    command -v docker >/dev/null 2>&1 || fail "docker is not installed"
    docker info >/dev/null 2>&1 ||
        fail "cannot talk to Docker — is the daemon running and are you in the docker group?"
}

compose() {
    local config_dir="$1"
    local container_user
    shift

    if [[ "$config_dir" == "$SYSTEM_CONFIG_DIR" ]]; then
        container_user="1000:1000"
    else
        container_user="$(id -u):$(id -g)"
    fi

    GITRAKZ_CONTAINER_USER="$container_user" docker compose \
        --project-directory "$config_dir" \
        --env-file "$config_dir/.env" \
        -f "$config_dir/docker-compose.yml" "$@"
}

ensure_data_dir() {
    local config_dir="$1" wrap="$2" container_user
    if [[ "$config_dir" == "$SYSTEM_CONFIG_DIR" ]]; then
        container_user="1000:1000"
    else
        container_user="$(id -u):$(id -g)"
    fi

    $wrap install -d -m 0700 "$config_dir/data"
    $wrap chown "$container_user" "$config_dir/data"
}

legacy_data_volume() {
    local config_dir="$1"

    [[ -f "$config_dir/docker-compose.yml" && -f "$config_dir/.env" ]] || return 0
    grep -Fq 'gitrakz-data:/data' "$config_dir/docker-compose.yml" || return 0

    GH_TOKEN=unused GITRAKZ_CONTAINER_USER=1000:1000 docker compose \
        --project-directory "$config_dir" \
        --env-file "$config_dir/.env" \
        -f "$config_dir/docker-compose.yml" config 2>/dev/null | awk '
            $0 == "volumes:" { in_volumes = 1; next }
            in_volumes && $0 == "  gitrakz-data:" { target = 1; next }
            target && $1 == "name:" { print $2; exit }
        '
}

migrate_legacy_data() {
    local config_dir="$1" wrap="$2" image="$3" volume_name="$4"
    local container_user
    [[ -n "$volume_name" && ! -e "$config_dir/data/gitrakz.db" ]] || return 0
    docker volume inspect "$volume_name" >/dev/null 2>&1 || return 0

    if [[ "$config_dir" == "$SYSTEM_CONFIG_DIR" ]]; then
        container_user="1000:1000"
    else
        container_user="$(id -u):$(id -g)"
    fi
    ensure_data_dir "$config_dir" "$wrap"
    say "moving existing gitrakz data into $config_dir/data"
    docker run --rm --user 0:0 --entrypoint sh \
        --mount "type=volume,source=$volume_name,target=/from,readonly" \
        --mount "type=bind,source=$config_dir/data,target=/to" \
        "$image" -ec 'cp -a /from/. /to/ && chown -R "$1" /to' sh "$container_user"
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
    local wrap
    wrap="$(root_wrap "$config_dir")"

    if grep -q "^${key}=" "$config_dir/.env"; then
        $wrap sed -i "s|^${key}=.*|${key}=${value}|" "$config_dir/.env"
    else
        $wrap bash -c 'printf "%s=%s\n" "$1" "$2" >>"$3"' _ "$key" "$value" "$config_dir/.env"
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
    local config_dir tag wrap legacy_volume
    require_docker
    config_dir="$(config_directory)"
    wrap="$(root_wrap "$config_dir")"
    $wrap mkdir -p "$config_dir"
    tag="$(resolve_latest_tag)"
    legacy_volume="$(legacy_data_volume "$config_dir")"
    ensure_data_dir "$config_dir" "$wrap"
    docker pull "$IMAGE_REPO:$tag"
    migrate_legacy_data "$config_dir" "$wrap" "$IMAGE_REPO:$tag" "$legacy_volume"

    $wrap curl -fsSL "$RAW_BASE/$tag/docker-compose.yml" \
        -o "$config_dir/docker-compose.yml"
    if [[ ! -f "$config_dir/.env" ]]; then
        $wrap curl -fsSL "$RAW_BASE/$tag/.env.example" -o "$config_dir/.env"
        $wrap chmod 600 "$config_dir/.env"
        env_set "$config_dir" GITRAKZ_IMAGE "$IMAGE_REPO:$tag"
    fi

    # Keep the shared-stack model when setup re-runs against system config.
    if [[ -n "$wrap" ]] && getent group docker >/dev/null 2>&1; then
        $wrap find "$config_dir" -path "$config_dir/data" -prune -o -exec chgrp docker {} + || true
        $wrap find "$config_dir" -path "$config_dir/data" -prune -o -type d -exec chmod g+rx {} + || true
        $wrap find "$config_dir" -path "$config_dir/data" -prune -o -type f -exec chmod g+r {} + || true
    fi

    say "config ready at $config_dir"
}

start() {
    require_docker
    local config_dir
    config_dir="$(config_directory)"
    ensure_config "$config_dir"

    export GH_TOKEN
    GH_TOKEN="$(gh_token)"

    if [[ "${GITRAKZ_ROLLING:-}" == "1" ]]; then
        warn "running the rolling :latest image for this run"
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
    local config_dir wrap
    config_dir="$(config_directory)"
    ensure_config "$config_dir"
    wrap="$(root_wrap "$config_dir")"

    if [[ "${GITRAKZ_ROLLING:-}" == "1" ]]; then
        warn "pulling the rolling :latest image (pin unchanged)"
        export GITRAKZ_IMAGE="$ROLLING_IMAGE"
        export GH_TOKEN
        GH_TOKEN="$(gh_token)"
        docker pull "$ROLLING_IMAGE"
        compose "$config_dir" up --detach
        compose "$config_dir" ps

        return
    fi

    local old_image new_tag new_image legacy_volume
    old_image="$(env_get "$config_dir" GITRAKZ_IMAGE)"
    new_tag="$(resolve_latest_tag)"
    new_image="$IMAGE_REPO:$new_tag"
    legacy_volume="$(legacy_data_volume "$config_dir")"

    if [[ "$old_image" == "$new_image" ]]; then
        say "already on the latest release $new_tag; re-pulling"
    fi

    docker pull "$new_image"
    ensure_data_dir "$config_dir" "$wrap"
    migrate_legacy_data "$config_dir" "$wrap" "$new_image" "$legacy_volume"
    $wrap curl -fsSL "$RAW_BASE/$new_tag/docker-compose.yml" \
        -o "$config_dir/docker-compose.yml"
    env_set "$config_dir" GITRAKZ_IMAGE "$new_image"

    export GH_TOKEN
    GH_TOKEN="$(gh_token)"
    compose "$config_dir" up --detach
    compose "$config_dir" ps

    # Refresh the wrapper itself from the new installer (self-update), re-running
    # in the same mode this install used.
    say "refreshing the installer + command"
    if [[ "$config_dir" == "$SYSTEM_CONFIG_DIR" ]]; then
        curl -fsSL "$INSTALLER_URL" | sudo bash -s -- --system
    else
        curl -fsSL "$INSTALLER_URL" | bash -s -- --user
    fi

    # Reclaim space: drop the previous image once the new one is running.
    if [[ -n "$old_image" && "$old_image" != "$new_image" ]]; then
        if docker image rm "$old_image" >/dev/null 2>&1; then
            say "removed the previous image $old_image"
        else
            warn "kept $old_image (still in use); prune it later for the space"
        fi
    fi
}

uninstall() {
    require_docker
    local config_dir answer delete_data=0 cmd_path wrap
    config_dir="$(config_directory)"
    wrap="$(root_wrap "$config_dir")"

    read -r -p "Also delete your data under $config_dir? [y/N] " answer
    case "$answer" in
    y | Y | yes | YES) delete_data=1 ;;
    esac

    if [[ -f "$config_dir/docker-compose.yml" ]]; then
        say "stopping the stack"
        if ((delete_data)); then
            compose "$config_dir" down --remove-orphans || true
        else
            compose "$config_dir" down --remove-orphans || true
        fi
    fi

    cmd_path="$(command -v gitrakz 2>/dev/null || printf '%s' "${BASH_SOURCE[0]}")"
    say "removing the gitrakz command ($cmd_path)"
    if [[ -w "$(dirname "$cmd_path")" ]]; then
        rm -f "$cmd_path"
    else
        command -v sudo >/dev/null || fail "removing $cmd_path needs root but sudo is not available"
        sudo rm -f "$cmd_path"
    fi

    if ((delete_data)); then
        $wrap rm -rf -- "$config_dir"
        say "deleted $config_dir and its data"
    else
        say "kept your data at $config_dir"
    fi

    say "gitrakz uninstalled"
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
SCRIPT

	install -m 0755 "$WRAPPER_TEMPORARY_FILE" "$INSTALL_PATH"
	say "installed the gitrakz command at $INSTALL_PATH"
}

main() {
	local arg
	for arg in "$@"; do
		case "$arg" in
		--system) MODE="system" ;;
		--user) MODE="user" ;;
		*) fail "unknown argument: $arg (supported: --system, --user)" ;;
		esac
	done

	resolve_mode

	docker info >/dev/null 2>&1 ||
		fail "cannot talk to Docker — is the daemon running, and (per-user) is your account in the docker group?"

	ensure_gh

	local previous_image new_tag new_image legacy_volume
	previous_image="$(env_pinned_image "$TARGET_CONFIG_DIR/.env")"
	new_tag="$(resolve_latest_tag)"
	new_image="$IMAGE_REPO:$new_tag"

	say "installing gitrakz ($MODE mode), pinning to the latest release $new_tag"
	prepare_config_dir
	legacy_volume="$(legacy_data_volume "$TARGET_CONFIG_DIR")"
	prepare_data_dir
	say "pulling $new_image"
	docker pull "$new_image"
	migrate_legacy_data "$legacy_volume" "$new_image"
	write_config "$new_tag"
	apply_config_permissions
	write_command
	warn_user_path

	if [[ -n "$previous_image" && "$previous_image" != "$new_image" ]]; then
		if docker image rm "$previous_image" >/dev/null 2>&1; then
			say "removed the previous image $previous_image"
		else
			warn "kept the previous image $previous_image (still in use)"
		fi
	fi

	printf '\ngitrakz is installed in %s mode (pinned to %s).\n\n' "$MODE" "$new_tag"
	printf '  Config:  %s/.env\n' "$TARGET_CONFIG_DIR"
	printf '  Command: %s\n\n' "$INSTALL_PATH"
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
