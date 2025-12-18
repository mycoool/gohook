#!/usr/bin/env bash
set -euo pipefail

REPO="${GOHOOK_REPO:-mycoool/gohook}"
GITHUB_API_BASE="https://api.github.com/repos/${REPO}"

PORT="${GOHOOK_PORT:-9000}"
PANEL_ALIAS="${GOHOOK_PANEL_ALIAS:-GoHook}"
INSTALL_SYSTEMD="${GOHOOK_INSTALL_SYSTEMD:-1}"
OVERWRITE_CONFIG="${GOHOOK_OVERWRITE_CONFIG:-0}"

ADMIN_USER="${GOHOOK_ADMIN_USER:-admin}"
ADMIN_PASSWORD="${GOHOOK_ADMIN_PASSWORD:-}"

log() { printf '%s\n' "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

need_cmd() {
	command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

as_root() {
	if [[ "$(id -u)" -eq 0 ]]; then
		"$@"
		return
	fi
	if command -v sudo >/dev/null 2>&1; then
		sudo "$@"
		return
	fi
	die "need root privileges to run: $* (install sudo or run as root)"
}

tmp_dir=""
cleanup() {
	[[ -n "${tmp_dir}" && -d "${tmp_dir}" ]] && rm -rf "${tmp_dir}"
}
trap cleanup EXIT

detect_arch() {
	local m
	m="$(uname -m)"
	case "${m}" in
		x86_64|amd64) echo "amd64" ;;
		aarch64|arm64) echo "arm64" ;;
		*) die "unsupported arch: ${m} (supported: x86_64/amd64, aarch64/arm64)" ;;
	esac
}

rand_str() {
	if command -v openssl >/dev/null 2>&1; then
		openssl rand -hex 16
		return
	fi
	# fallback
	LC_ALL=C tr -dc 'a-zA-Z0-9' </dev/urandom | head -c 32
}

sha256_hex() {
	local s="${1}"
	if command -v sha256sum >/dev/null 2>&1; then
		printf '%s' "${s}" | sha256sum | awk '{print $1}'
		return
	fi
	if command -v shasum >/dev/null 2>&1; then
		printf '%s' "${s}" | shasum -a 256 | awk '{print $1}'
		return
	fi
	die "missing sha256 tool (need sha256sum or shasum)"
}

extract_zip() {
	local zip_path="${1}"
	local out_dir="${2}"
	mkdir -p "${out_dir}"
	if command -v unzip >/dev/null 2>&1; then
		unzip -q -o "${zip_path}" -d "${out_dir}"
		return
	fi
	if command -v python3 >/dev/null 2>&1; then
		python3 - "${zip_path}" "${out_dir}" <<'PY'
import sys, zipfile
zip_path, out_dir = sys.argv[1], sys.argv[2]
with zipfile.ZipFile(zip_path, 'r') as z:
    z.extractall(out_dir)
PY
		return
	fi
	die "missing unzip (need unzip or python3)"
}

github_latest_asset_url() {
	local want_name="${1}"
	local json
	json="$(curl -fsSL "${GITHUB_API_BASE}/releases/latest")"

	if command -v python3 >/dev/null 2>&1; then
		python3 - "${want_name}" <<'PY'
import json, sys
want = sys.argv[1]
data = json.load(sys.stdin)
for asset in data.get("assets", []):
    if asset.get("name") == want:
        print(asset.get("browser_download_url") or "")
        raise SystemExit(0)
raise SystemExit(2)
PY
		return
	fi

	# fallback: naive parse without jq/python
	printf '%s\n' "${json}" \
		| awk -v want="\"name\": \"${want_name}\"" '
			$0 ~ want {found=1}
			found && $0 ~ /browser_download_url/ {
				gsub(/.*"browser_download_url": "/, "", $0)
				gsub(/".*/, "", $0)
				print $0
				exit 0
			}
		' || true
}

write_file_if_needed() {
	local path="${1}"
	local mode="${2}"
	local content="${3}"
	if [[ -f "${path}" && "${OVERWRITE_CONFIG}" != "1" ]]; then
		return 0
	fi
	umask 077
	printf '%s' "${content}" | as_root tee "${path}" >/dev/null
	as_root chmod "${mode}" "${path}"
}

main() {
	need_cmd curl

	[[ "$(uname -s)" == "Linux" ]] || die "this installer only supports Linux"

	local arch asset url
	arch="$(detect_arch)"
	asset="gohook-linux-${arch}.zip"
	url="$(github_latest_asset_url "${asset}")"
	[[ -n "${url}" ]] || die "could not find release asset: ${asset}"

	tmp_dir="$(mktemp -d)"
	log "Downloading ${REPO} latest release: ${asset}"
	curl -fL "${url}" -o "${tmp_dir}/${asset}"

	extract_zip "${tmp_dir}/${asset}" "${tmp_dir}/out"
	[[ -f "${tmp_dir}/out/gohook-linux-${arch}" ]] || die "unexpected zip layout: missing gohook-linux-${arch}"

	local bin_dir config_dir data_dir log_dir
	if [[ "$(id -u)" -eq 0 ]]; then
		bin_dir="${GOHOOK_INSTALL_BIN_DIR:-/usr/local/bin}"
		config_dir="${GOHOOK_CONFIG_DIR:-/etc/gohook}"
		data_dir="${GOHOOK_DATA_DIR:-/var/lib/gohook}"
		log_dir="${GOHOOK_LOG_DIR:-/var/log/gohook}"
	else
		bin_dir="${GOHOOK_INSTALL_BIN_DIR:-${HOME}/.local/bin}"
		config_dir="${GOHOOK_CONFIG_DIR:-${HOME}/.config/gohook}"
		data_dir="${GOHOOK_DATA_DIR:-${HOME}/.local/share/gohook}"
		log_dir="${GOHOOK_LOG_DIR:-${HOME}/.local/state/gohook}"
		INSTALL_SYSTEMD="0"
	fi

	as_root mkdir -p "${bin_dir}" "${config_dir}" "${data_dir}" "${log_dir}"
	as_root install -m 0755 "${tmp_dir}/out/gohook-linux-${arch}" "${bin_dir}/gohook"

	# configs (all loaded relative to working directory)
	local jwt_secret
	jwt_secret="$(rand_str)"

	write_file_if_needed "${config_dir}/hooks.json" 0644 "[]\n"
	write_file_if_needed "${config_dir}/version.yaml" 0644 "projects: []\n"

	local admin_password admin_password_hash
	if [[ ! -f "${config_dir}/user.yaml" || "${OVERWRITE_CONFIG}" == "1" ]]; then
		admin_password="${ADMIN_PASSWORD:-$(rand_str)}"
		admin_password_hash="$(sha256_hex "${admin_password}")"
		write_file_if_needed "${config_dir}/user.yaml" 0600 \
			"users:\n  - username: ${ADMIN_USER}\n    password: ${admin_password_hash}\n    role: admin\n"
		log "Created ${config_dir}/user.yaml"
		log "Admin login: ${ADMIN_USER}"
		log "Admin password: ${admin_password}"
	else
		log "Keeping existing ${config_dir}/user.yaml (set GOHOOK_OVERWRITE_CONFIG=1 to overwrite)"
	fi

	if [[ ! -f "${config_dir}/app.yaml" || "${OVERWRITE_CONFIG}" == "1" ]]; then
		write_file_if_needed "${config_dir}/app.yaml" 0644 \
			"port: ${PORT}\nmode: prod\npanel_alias: ${PANEL_ALIAS}\njwt_secret: ${jwt_secret}\njwt_expiry_duration: 1440\ndatabase:\n  type: sqlite\n  database: ${data_dir}/gohook.db\n  log_retention_days: 30\n"
		log "Created ${config_dir}/app.yaml"
	else
		log "Keeping existing ${config_dir}/app.yaml (set GOHOOK_OVERWRITE_CONFIG=1 to overwrite)"
	fi

	if [[ "${INSTALL_SYSTEMD}" == "1" ]] && command -v systemctl >/dev/null 2>&1; then
		local svc_path="/etc/systemd/system/gohook.service"

		if ! id -u gohook >/dev/null 2>&1; then
			if command -v useradd >/dev/null 2>&1; then
				as_root useradd --system --home "${data_dir}" --shell /usr/sbin/nologin gohook || true
			fi
		fi

		if id -u gohook >/dev/null 2>&1; then
			as_root chown -R gohook:gohook "${config_dir}" "${data_dir}" "${log_dir}"
		fi

		if [[ ! -f "${svc_path}" || "${OVERWRITE_CONFIG}" == "1" ]]; then
			local run_user="root"
			local run_group="root"
			if id -u gohook >/dev/null 2>&1; then
				run_user="gohook"
				run_group="gohook"
			fi

			as_root tee "${svc_path}" >/dev/null <<EOF
[Unit]
Description=GoHook
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${config_dir}
ExecStart=${bin_dir}/gohook -ip 0.0.0.0 -port ${PORT} -hooks hooks.json -logfile ${log_dir}/gohook.log
Restart=on-failure
User=${run_user}
Group=${run_group}
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOF
			log "Wrote ${svc_path}"
		else
			log "Keeping existing ${svc_path} (set GOHOOK_OVERWRITE_CONFIG=1 to overwrite)"
		fi

		as_root systemctl daemon-reload
		as_root systemctl enable --now gohook
		log "systemd: enabled and started gohook"
	else
		log "Install complete."
		log "Run manually:"
		log "  cd ${config_dir} && ${bin_dir}/gohook -ip 0.0.0.0 -port ${PORT}"
		if [[ "$(id -u)" -ne 0 ]]; then
			log "Ensure ${bin_dir} is in your PATH."
		fi
	fi
}

main "$@"

