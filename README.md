# fcitx5-lotus-installer

Trình cài đặt tự động cho bộ gõ tiếng Việt **fcitx5-lotus** trên Linux.

## 🚀 Cài đặt nhanh

```bash
curl -fsSL https://raw.githubusercontent.com/hthienloc/fcitx5-lotus-installer/main/install.sh | sh
```

Hoặc chạy với chế độ CLI (không TUI):

```bash
curl -fsSL https://raw.githubusercontent.com/hthienloc/fcitx5-lotus-installer/main/install.sh | sh -s -- --cli
```

## ✨ Tính năng

- **Auto-detect OS**: Arch, Debian, Ubuntu, Fedora, openSUSE
- **Auto-check dependencies**: Kiểm tra và cài đặt các package cần thiết
- **Build from source**: Clone, cấu hình cmake, build và install tự động
- **Post-install configure**: Tự động thiết lập environment variables và fcitx5 profile
- **TUI interactive**: Giao diện terminal đẹp với step-by-step progress
- **CLI mode**: Chế độ dòng lệnh cho CI/automation

## 🖥️ Distro hỗ trợ

| Distro | Package Manager | Notes |
|--------|----------------|-------|
| Arch / Manjaro / EndeavourOS | pacman | Official repos + AUR |
| Debian / Linux Mint | apt | Debian 12+ |
| Ubuntu / Pop!_OS | apt | Ubuntu 22.04+ |
| Fedora / Nobara | dnf | Fedora 38+ |
| openSUSE Tumbleweed | zypper | Rolling release |

## 🏗️ Cấu trúc dự án

```
fcitx5-lotus-installer/
├── cmd/installer/main.go      # Entry point (TUI + CLI)
├── internal/
│   ├── distro/distro.go       # OS detection logic
│   ├── packages/packages.go   # Dependency management
│   ├── build/build.go         # CMake build logic
│   └── configure/configure.go # Post-install configuration
├── install.sh                 # Bootstrap script
├── go.mod
└── README.md
```

## 🛠️ Phát triển

### Yêu cầu

- Go 1.21+

### Build

```bash
git clone https://github.com/hthienloc/fcitx5-lotus-installer.git
cd fcitx5-lotus-installer
go build -o lotus-installer ./cmd/installer/
```

### Run

```bash
# TUI mode (interactive)
./lotus-installer

# CLI mode (non-interactive)
./lotus-installer --cli
```

### Build for release

```bash
GOOS=linux GOARCH=amd64 go build -o lotus-installer-linux-amd64 ./cmd/installer/
GOOS=linux GOARCH=arm64 go build -o lotus-installer-linux-arm64 ./cmd/installer/
```

## 📝 Sau khi cài đặt

1. Log out và log back in (hoặc restart)
2. Mở `fcitx5-configtool` và thêm input method "Lotus"
3. Bắt đầu gõ tiếng Việt!

Restart thủ công nếu cần:

```bash
fcitx5 -d --replace
```

## 📖 Tài liệu

- Website: https://lotusinputmethod.github.io/
- Source: https://github.com/LotusInputMethod/fcitx5-lotus

## 📄 License

GPL-3.0
