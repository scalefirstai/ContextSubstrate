#!/usr/bin/env bash
set -euo pipefail

# ContextSubstrate installer
# Usage: curl -fsSL https://raw.githubusercontent.com/scalefirstai/ContextSubstrate/main/scripts/install.sh | bash

REPO="scalefirstai/ContextSubstrate"
BINARY="ctx"
INSTALL_DIR="/usr/local/bin"

info()  { printf "\033[1;34m==>\033[0m %s\n" "$*"; }
error() { printf "\033[1;31merror:\033[0m %s\n" "$*" >&2; exit 1; }

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux*)  OS="linux" ;;
  Darwin*) OS="darwin" ;;
  *)       error "Unsupported OS: $OS" ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  amd64)   ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       error "Unsupported architecture: $ARCH" ;;
esac

# Get latest release tag
info "Fetching latest release..."
TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')"
[ -n "$TAG" ] || error "Could not determine latest release"
VERSION="${TAG#v}"

info "Installing ${BINARY} ${TAG} (${OS}/${ARCH})"

# Download archive
ARCHIVE="ContextSubstrate_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading ${URL}..."
curl -fsSL "$URL" -o "${TMPDIR}/${ARCHIVE}" || error "Download failed. Check that a release exists for ${OS}/${ARCH}."

# Extract
tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  info "Elevated permissions required to install to ${INSTALL_DIR}"
  sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"

info "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
echo ""
"${INSTALL_DIR}/${BINARY}" --version
echo ""
info "Run 'ctx init' in your project to get started."
