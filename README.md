# fcitx5-lotus-installer

Trình cài đặt tự động cho bộ gõ tiếng Việt **fcitx5-lotus** trên Linux.

## 🚀 Cài đặt nhanh

```bash
curl -fsSL https://raw.githubusercontent.com/hthienloc/fcitx5-lotus-installer/main/install.sh | sh
```

## ✨ Tính năng

- **Auto-detect OS**: Arch, Debian, Ubuntu, Fedora, openSUSE, Void Linux, NixOS
- **Auto-detect init system**: systemd, OpenRC, runit
- **Auto-detect shell**: Bash, Zsh, Fish
- **Auto-detect session**: X11, Wayland
- **Auto-check dependencies**: Kiểm tra và cài đặt các package cần thiết
- **Build from source**: Clone, cấu hình cmake, build và install tự động
- **Post-install services**:
  - Tạo `uinput_proxy` user/group
  - Reload udev rules
  - Modprobe uinput kernel module
  - Kích hoạt `fcitx5-lotus-server` service
  - Kill IBus để tránh xung đột
- **Post-install configure**:
  - Environment variables (DE-specific cho Wayland)
  - Shell profile config (Bash/Zsh/Fish)
  - Fcitx5 profile với Lotus input method
  - Autostart setup cho DE/WM
- **Wayland support**: KDE Plasma, GNOME, Sway-specific env vars và Chromium flags

## 🖥️ Distro hỗ trợ

| Distro | Package Manager | Init Systems | Notes |
|--------|----------------|-------------|-------|
| Arch / Manjaro / EndeavourOS | pacman | systemd | AUR: `fcitx5-lotus-bin`, `fcitx5-lotus-git` |
| Debian / Linux Mint | apt | systemd | APT repo qua fcitx5-lotus.pages.dev |
| Ubuntu / Pop!_OS | apt | systemd | APT repo qua fcitx5-lotus.pages.dev |
| Fedora / Nobara | dnf | systemd | RPM repo qua fcitx5-lotus.pages.dev |
| openSUSE Tumbleweed | zypper | systemd | RPM repo qua fcitx5-lotus.pages.dev |
| Void Linux | xbps | runit | Build from source (chưa có package chính thức) |
| NixOS | nix flakes | systemd | Flake/module method |

## 🖥️ DE/WM hỗ trợ

GNOME, KDE Plasma, Xfce, Cinnamon, MATE, Pantheon, Budgie, LXQt, COSMIC, i3, Sway, Hyprland

## 🏗️ Cấu trúc dự án

```
fcitx5-lotus-installer/
├── cmd/installer/main.go        # Entry point - interactive CLI
├── internal/
│   ├── distro/distro.go         # OS detection (7 distros)
│   ├── packages/packages.go     # Dependency management
│   ├── build/build.go           # CMake build logic
│   ├── services/services.go     # Server, uinput, udev, IBus
│   └── configure/configure.go   # Env vars, shell, autostart
├── install.sh                   # Bootstrap script
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
./lotus-installer
```

### Build for release

```bash
GOOS=linux GOARCH=amd64 go build -o lotus-installer-linux-amd64 ./cmd/installer/
GOOS=linux GOARCH=arm64 go build -o lotus-installer-linux-arm64 ./cmd/installer/
```

## 📋 Installer workflow

1. **Detect OS** — đọc `/etc/os-release`, phân loại distro
2. **Detect init system** — systemd/OpenRC/runit
3. **Detect shell** — $SHELL env (Bash/Zsh/Fish)
4. **Select DE** — user chọn từ menu
5. **Detect session** — $XDG_SESSION_TYPE (X11/Wayland)
6. **Select method** — Package Manager hoặc Build from Source
7. **Install deps** — chỉ cài những package thiếu
8. **Clone & build** — từ GitHub source
9. **Post-install services**:
   - Tạo `uinput_proxy` user/group
   - `udevadm control --reload-rules`
   - `modprobe uinput`
   - `systemctl enable --now fcitx5-lotus-server@$(whoami)`
10. **Kill IBus** — tránh xung đột
11. **Configure**:
    - `~/.config/environment.d/90-fcitx5-lotus.conf`
    - Shell profile (`.bash_profile`, `.zprofile`, hoặc `config.fish`)
    - Fcitx5 profile với Lotus làm default IM
    - Autostart desktop file + DE-specific config (i3, Sway, Hyprland)
12. **Restart fcitx5** — `fcitx5 -d --replace`

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
