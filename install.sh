#!/usr/bin/env bash

# fcitx5-lotus One-Liner Installer Bootstrap
# Usage: curl -fsSL https://raw.githubusercontent.com/hthienloc/fcitx5-lotus-installer/main/install.sh | sh

set -e

# Colors for terminal
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}🪷  Initializing fcitx5-lotus Installer...${NC}"

# 1. Root check
if [ "$EUID" -eq 0 ]; then
  echo -e "${RED}Error: Please do not run as root.${NC}"
  echo "The installer will ask for sudo when needed."
  exit 1
fi

# 2. Architecture detection
ARCH=$(uname -m)
case $ARCH in
    x86_64)  BINARY_ARCH="amd64" ;;
    aarch64) BINARY_ARCH="arm64" ;;
    *) echo -e "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

# 3. OS detection
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS_NAME=$NAME
    OS_VERSION=$VERSION_ID
    echo -e "${BLUE}📋 Detected: $OS_NAME $OS_VERSION ($ARCH)${NC}"
fi

# 4. Check if Go is available for building from source
if command -v go &> /dev/null; then
    echo -e "${GREEN}✅ Go found, building installer from source...${NC}"
    TEMP_DIR=$(mktemp -d)
    git clone --depth 1 https://github.com/hthienloc/fcitx5-lotus-installer.git "$TEMP_DIR" 2>/dev/null || {
        echo -e "${YELLOW}⚠️  Git clone failed, trying direct download...${NC}"
    }

    if [ -d "$TEMP_DIR/cmd/installer" ]; then
        cd "$TEMP_DIR"
        go build -o lotus-installer ./cmd/installer/
        chmod +x lotus-installer
        echo -e "${GREEN}✅ Built successfully${NC}"
        ./lotus-installer
        rm -rf "$TEMP_DIR"
        exit $?
    fi
fi

# 5. Fallback: download pre-built binary
INSTALLER_BIN="/tmp/lotus-installer"
echo -e "${BLUE}📥 Downloading installer for ${BINARY_ARCH}...${NC}"

GITHUB_API="https://api.github.com/repos/hthienloc/fcitx5-lotus-installer/releases/latest"
LATEST_TAG=$(curl -fsSL "$GITHUB_API" 2>/dev/null | grep -o '"tag_name": *"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$LATEST_TAG" ]; then
    DOWNLOAD_URL="https://github.com/hthienloc/fcitx5-lotus-installer/releases/download/$LATEST_TAG/lotus-installer-linux-$BINARY_ARCH"
    curl -fsSL -o "$INSTALLER_BIN" "$DOWNLOAD_URL" || {
        echo -e "${RED}❌ Download failed${NC}"
        echo -e "${YELLOW}Fallback: Install Go and run:${NC}"
        echo "  go install github.com/hthienloc/fcitx5-lotus-installer/cmd/installer@latest"
        exit 1
    }
    chmod +x "$INSTALLER_BIN"
    "$INSTALLER_BIN"
else
    echo -e "${RED}❌ Could not determine latest release${NC}"
    echo -e "${YELLOW}Please install Go and build manually:${NC}"
    echo "  git clone https://github.com/hthienloc/fcitx5-lotus-installer.git"
    echo "  cd fcitx5-lotus-installer"
    echo "  go build -o lotus-installer ./cmd/installer/"
    echo "  ./lotus-installer"
    exit 1
fi
