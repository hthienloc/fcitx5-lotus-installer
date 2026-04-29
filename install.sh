#!/usr/bin/env bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

printf "\n${BOLD}${BLUE}  🪷  fcitx5-lotus Installer${NC}\n"
printf "${BLUE}  ───────────────────────────────${NC}\n\n"

# Root check
if [ "$(id -u)" = "0" ]; then
    printf "${RED}Error: Do not run as root.${NC}\n"
    echo "The installer will ask for sudo when needed."
    exit 1
fi

# Linux check
if [ "$(uname)" != "Linux" ]; then
    printf "${RED}Error: This installer only supports Linux.${NC}\n"
    exit 1
fi

# Detect arch
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  BIN_ARCH="amd64" ;;
    aarch64) BIN_ARCH="arm64" ;;
    *) printf "${RED}Unsupported architecture: %s${NC}\n" "$ARCH"; exit 1 ;;
esac

printf "${GREEN}Detecting system...${NC} %s (%s)\n\n" "$(uname -o 2>/dev/null || echo Linux)" "$ARCH"

# Try to download pre-built binary first
LATEST_VERSION=$(curl -sf https://api.github.com/repos/hthienloc/fcitx5-lotus-installer/releases/latest 2>/dev/null | grep -o '"tag_name": *"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$LATEST_VERSION" ]; then
    printf "${GREEN}Found release ${BOLD}%s${NC}\n" "$LATEST_VERSION"
    printf "${GREEN}Downloading installer...${NC}\n"

    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"

    curl -sfL "https://github.com/hthienloc/fcitx5-lotus-installer/releases/download/$LATEST_VERSION/lotus-installer-linux-$BIN_ARCH" -o lotus-installer 2>/dev/null

    if [ -f lotus-installer ]; then
        chmod +x lotus-installer
        printf "${GREEN}Running installer...${NC}\n\n"
        ./lotus-installer
        cd - >/dev/null
        rm -rf "$TEMP_DIR"
        exit 0
    fi

    cd - >/dev/null
    rm -rf "$TEMP_DIR"
    printf "${YELLOW}No pre-built binary for %s, building from source...${NC}\n\n" "$BIN_ARCH"
fi

# Fallback: build from source
if ! command -v go &> /dev/null; then
    printf "${RED}Error: Go is required to build the installer.${NC}\n"
    echo "Install Go or download a pre-built binary from GitHub releases."
    exit 1
fi

printf "${GREEN}Go found. Building installer...${NC}\n"

TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

git clone --depth 1 -q https://github.com/hthienloc/fcitx5-lotus-installer.git . 2>/dev/null
go build -o lotus-installer ./cmd/installer/ 2>/dev/null
chmod +x lotus-installer

printf "${GREEN}Running installer...${NC}\n\n"
./lotus-installer

cd - >/dev/null
rm -rf "$TEMP_DIR"
