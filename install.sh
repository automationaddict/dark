#!/usr/bin/env bash

# dark installer — user-scope. Downloads the latest release (or a
# specific --version), verifies the SHA256 checksum, and installs
# the three binaries under ~/.local/bin alongside a systemd user
# unit at ~/.config/systemd/user/darkd.service. Nothing outside
# $HOME is touched; darkd delegates every privileged operation to
# dark-helper at runtime via pkexec.
#
# Typical one-liner:
#
#   curl -fsSL https://raw.githubusercontent.com/automationaddict/dark/main/install.sh | bash
#
# Or to pin a specific version:
#
#   curl -fsSL https://raw.githubusercontent.com/automationaddict/dark/main/install.sh | bash -s -- --version v0.1.0
#
# The in-app self-update flow also calls this script via the same
# subprocess path so there's a single code path for both first
# install and update.

set -euo pipefail

# ─── configuration ────────────────────────────────────────────────

REPO="automationaddict/dark"
BINARIES=(dark darkd dark-helper)
PREFIX_BIN="${HOME}/.local/bin"
UNIT_DIR="${HOME}/.config/systemd/user"
CACHE_DIR="${XDG_CACHE_HOME:-${HOME}/.cache}/dark/install"

# Parsed arguments — empty version means "latest".
VERSION=""
SKIP_UNIT=0
SKIP_ENABLE=0
FORCE=0

# ─── pretty output ────────────────────────────────────────────────

bold() { printf '\033[1m%s\033[0m\n' "$*"; }
dim()  { printf '\033[2m%s\033[0m\n' "$*"; }
err()  { printf '\033[31merror:\033[0m %s\n' "$*" >&2; }
ok()   { printf '\033[32m✓\033[0m %s\n' "$*"; }

usage() {
  cat <<USAGE
usage: install.sh [options]

Options:
  --version <tag>    Install a specific release tag (e.g. v0.1.0).
                     Defaults to the latest release.
  --skip-unit        Don't install the systemd user unit.
  --skip-enable      Install the unit but don't enable/start it.
  --force            Reinstall even if the requested version is
                     already present.
  -h, --help         Show this help.

The installer always writes to:
  ~/.local/bin/{dark,darkd,dark-helper}
  ~/.config/systemd/user/darkd.service

No sudo required — every privileged operation is delegated to
dark-helper at runtime via pkexec.
USAGE
}

# ─── argument parsing ─────────────────────────────────────────────

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      shift
      [[ $# -gt 0 ]] || { err "--version requires a value"; exit 2; }
      VERSION="$1"
      ;;
    --version=*)
      VERSION="${1#*=}"
      ;;
    --skip-unit)
      SKIP_UNIT=1
      ;;
    --skip-enable)
      SKIP_ENABLE=1
      ;;
    --force)
      FORCE=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      err "unknown option: $1"
      usage >&2
      exit 2
      ;;
  esac
  shift
done

# ─── dependency check ─────────────────────────────────────────────

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    err "required command '$1' is missing — please install it and re-run"
    exit 1
  }
}

require_cmd curl
require_cmd tar
require_cmd sha256sum
require_cmd uname

# ─── arch check ───────────────────────────────────────────────────
# dark currently only publishes linux/amd64 binaries. Fail fast on
# anything else so users on ARM don't get a cryptic download error.

UNAME_S=$(uname -s)
UNAME_M=$(uname -m)
if [[ "$UNAME_S" != "Linux" ]]; then
  err "dark only runs on Linux (detected $UNAME_S)"
  exit 1
fi
if [[ "$UNAME_M" != "x86_64" ]]; then
  err "dark currently only ships linux/amd64 binaries (detected $UNAME_M)"
  err "build from source: git clone https://github.com/${REPO} && cd dark && go build ./cmd/..."
  exit 1
fi

# ─── resolve version ──────────────────────────────────────────────

fetch_latest_tag() {
  # GitHub's redirect-based latest endpoint works without the API
  # token so anonymous installs don't hit rate limits as fast.
  # The Location header contains the canonical tag.
  local url
  url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
    "https://github.com/${REPO}/releases/latest") || return 1
  echo "${url##*/}"
}

if [[ -z "$VERSION" ]]; then
  bold "Looking up latest dark release…"
  VERSION=$(fetch_latest_tag) || {
    err "couldn't resolve latest release — check that the repo has published releases"
    exit 1
  }
fi

# Normalize — accept both `v0.1.0` and `0.1.0`.
case "$VERSION" in
  v*) TAG="$VERSION" ;;
  *)  TAG="v$VERSION" ;;
esac

bold "Installing dark $TAG"

# ─── already-installed short-circuit ──────────────────────────────

if [[ $FORCE -eq 0 && -x "${PREFIX_BIN}/dark" ]]; then
  CURRENT=$("${PREFIX_BIN}/dark" --version 2>/dev/null || echo "")
  if [[ -n "$CURRENT" && "$CURRENT" == *"${TAG#v}"* ]]; then
    ok "dark ${TAG} already installed at ${PREFIX_BIN}/dark"
    dim "   use --force to reinstall"
    exit 0
  fi
fi

# ─── download ─────────────────────────────────────────────────────

TARBALL="dark-${TAG}-linux-amd64.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

mkdir -p "$CACHE_DIR"
trap 'rm -rf "$CACHE_DIR"' EXIT

bold "Downloading ${TARBALL}"
curl -fsSL -o "${CACHE_DIR}/${TARBALL}" "${BASE_URL}/${TARBALL}" || {
  err "failed to download ${BASE_URL}/${TARBALL}"
  err "check the release exists: https://github.com/${REPO}/releases/tag/${TAG}"
  exit 1
}
ok "downloaded ${TARBALL}"

bold "Downloading SHA256SUMS"
curl -fsSL -o "${CACHE_DIR}/SHA256SUMS" "${BASE_URL}/SHA256SUMS" || {
  err "failed to download SHA256SUMS"
  exit 1
}
ok "downloaded SHA256SUMS"

# ─── checksum verify ──────────────────────────────────────────────

bold "Verifying checksum"
(
  cd "$CACHE_DIR"
  # Filter SHA256SUMS to just the tarball we downloaded, in case
  # future releases ever bundle multiple files. sha256sum -c
  # needs the filename present in the current directory.
  if ! grep -F "$TARBALL" SHA256SUMS | sha256sum -c --quiet --status; then
    err "SHA256 checksum mismatch — download may be corrupted or tampered with"
    exit 1
  fi
)
ok "checksum matches"

# ─── extract ──────────────────────────────────────────────────────

bold "Extracting to ${CACHE_DIR}/stage"
mkdir -p "${CACHE_DIR}/stage"
tar -C "${CACHE_DIR}/stage" -xzf "${CACHE_DIR}/${TARBALL}"

for bin in "${BINARIES[@]}"; do
  if [[ ! -x "${CACHE_DIR}/stage/${bin}" ]]; then
    err "tarball missing expected binary: ${bin}"
    exit 1
  fi
done
ok "extracted ${#BINARIES[@]} binaries"

# ─── install ──────────────────────────────────────────────────────

mkdir -p "$PREFIX_BIN"
bold "Installing binaries to ${PREFIX_BIN}"
for bin in "${BINARIES[@]}"; do
  # Atomic install via write-to-tmp + rename. Overwriting a
  # running binary is safe on Linux because the kernel holds a
  # reference via the mapped inode; the new file just takes the
  # path and the next exec picks it up.
  install -Dm 0755 "${CACHE_DIR}/stage/${bin}" "${PREFIX_BIN}/${bin}.new"
  mv -f "${PREFIX_BIN}/${bin}.new" "${PREFIX_BIN}/${bin}"
  ok "installed ${bin}"
done

# ─── systemd user unit ────────────────────────────────────────────

if [[ $SKIP_UNIT -eq 0 ]]; then
  bold "Installing systemd user unit"
  mkdir -p "$UNIT_DIR"

  # The unit ships inside the install script so the one-liner
  # works without needing a second download. Keep this in sync
  # with dist/systemd/darkd.service in the repo.
  cat > "${UNIT_DIR}/darkd.service" <<'UNIT'
[Unit]
Description=dark settings panel daemon
Documentation=https://github.com/automationaddict/dark
After=graphical-session.target
PartOf=graphical-session.target

[Service]
Type=simple
ExecStart=%h/.local/bin/darkd
Restart=on-failure
RestartSec=2s

NoNewPrivileges=yes
ProtectSystem=full
ProtectHome=read-only
RuntimeDirectory=dark
CacheDirectory=dark
ReadWritePaths=%h/.cache/dark
PrivateTmp=yes

[Install]
WantedBy=graphical-session.target
UNIT
  ok "wrote ${UNIT_DIR}/darkd.service"

  if [[ $SKIP_ENABLE -eq 0 ]] && command -v systemctl >/dev/null 2>&1; then
    systemctl --user daemon-reload
    if systemctl --user enable --now darkd.service 2>/dev/null; then
      ok "enabled and started darkd.service"
    else
      dim "   run 'systemctl --user enable --now darkd.service' once you're in a graphical session"
    fi
  fi
fi

# ─── PATH reminder ────────────────────────────────────────────────

case ":$PATH:" in
  *":$PREFIX_BIN:"*) ;;
  *)
    echo
    dim "Note: ${PREFIX_BIN} is not in your PATH."
    dim "      Add it to your shell profile so you can run 'dark':"
    dim "        echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
    ;;
esac

# ─── done ─────────────────────────────────────────────────────────

echo
bold "✔ dark ${TAG} installed successfully"
echo
echo "Run: dark"
echo "Help: press ? inside dark"
echo "Releases: https://github.com/${REPO}/releases"
