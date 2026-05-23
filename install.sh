#!/usr/bin/env bash
# harness-sync installer
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/lukaszraczylo/harness-sync/main/install.sh | bash
# Optional:
#   INSTALL_DIR=/usr/local/bin curl ... | bash
#   VERSION=v1.2.3 curl ... | bash

set -euo pipefail

REPO="lukaszraczylo/harness-sync"
BINARY_NAME="harness-sync"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { printf '%b[INFO]%b %s\n' "$GREEN" "$NC" "$1"; }
warn()  { printf '%b[WARN]%b %s\n' "$YELLOW" "$NC" "$1"; }
error() { printf '%b[ERROR]%b %s\n' "$RED" "$NC" "$1" >&2; }

detect_platform() {
    local os arch
    case "$(uname -s)" in
        Linux*)   os="linux" ;;
        Darwin*)  os="darwin" ;;
        MINGW*|MSYS*|CYGWIN*)
            error "Use WSL or download the Windows zip from the releases page."
            exit 1
            ;;
        *)
            error "Unsupported OS: $(uname -s)"
            exit 1
            ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64)  arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *)
            error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    echo "${os}-${arch}"
}

check_deps() {
    local missing=()
    for cmd in curl tar; do
        command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd")
    done
    if [ ${#missing[@]} -ne 0 ]; then
        error "Missing required commands: ${missing[*]}"
        exit 1
    fi
}

resolve_version() {
    if [ -n "$VERSION" ]; then
        echo "$VERSION"
        return
    fi
    local tag
    tag=$(curl -sSfL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | head -1 \
        | cut -d'"' -f4)
    if [ -z "$tag" ]; then
        error "Could not resolve latest release tag from GitHub."
        exit 1
    fi
    echo "$tag"
}

download_and_extract() {
    local version="$1" platform="$2" tmpdir="$3"
    local archive="harness-sync-${platform}.tar.gz"
    local url="https://github.com/${REPO}/releases/download/${version}/${archive}"
    local checksums_url="https://github.com/${REPO}/releases/download/${version}/harness-sync-checksums.txt"

    info "Downloading ${archive} from ${version}..."
    if ! curl -sSfL "$url" -o "$tmpdir/$archive"; then
        error "Failed to download $url"
        exit 1
    fi

    info "Verifying checksum..."
    if curl -sSfL "$checksums_url" -o "$tmpdir/checksums.txt"; then
        local expected actual
        expected=$(grep " $archive\$" "$tmpdir/checksums.txt" | awk '{print $1}' || true)
        if [ -n "$expected" ]; then
            if command -v sha256sum >/dev/null 2>&1; then
                actual=$(sha256sum "$tmpdir/$archive" | awk '{print $1}')
            elif command -v shasum >/dev/null 2>&1; then
                actual=$(shasum -a 256 "$tmpdir/$archive" | awk '{print $1}')
            fi
            if [ -n "${actual:-}" ] && [ "$expected" != "$actual" ]; then
                error "Checksum mismatch (expected $expected, got $actual)"
                exit 1
            fi
        else
            warn "Checksum entry for $archive not found, skipping verification."
        fi
    else
        warn "Checksums file unavailable, skipping verification."
    fi

    tar -xzf "$tmpdir/$archive" -C "$tmpdir"
    if [ ! -f "$tmpdir/$BINARY_NAME" ]; then
        error "Archive did not contain a $BINARY_NAME binary."
        exit 1
    fi
}

install_binary() {
    local tmpdir="$1"

    if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
        warn "Cannot create $INSTALL_DIR; retrying with sudo..."
        sudo mkdir -p "$INSTALL_DIR"
    fi

    local dest="$INSTALL_DIR/$BINARY_NAME"
    if [ -w "$INSTALL_DIR" ]; then
        install -m 0755 "$tmpdir/$BINARY_NAME" "$dest"
    else
        sudo install -m 0755 "$tmpdir/$BINARY_NAME" "$dest"
    fi

    info "Installed $dest"

    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *)
            warn "$INSTALL_DIR is not on your PATH. Add this to your shell rc:"
            printf '  export PATH="%s:$PATH"\n' "$INSTALL_DIR"
            ;;
    esac
}

main() {
    check_deps
    local platform version tmpdir
    platform=$(detect_platform)
    version=$(resolve_version)
    info "harness-sync $version → $INSTALL_DIR ($platform)"

    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT
    download_and_extract "$version" "$platform" "$tmpdir"
    install_binary "$tmpdir"

    info "Done. Run: $BINARY_NAME --help"
}

main "$@"
