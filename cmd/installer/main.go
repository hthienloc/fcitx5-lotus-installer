package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/hthienloc/fcitx5-lotus-installer/internal/build"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/configure"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/distro"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/packages"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/services"
)

type InstallMethod string

const (
	PackageManager InstallMethod = "Package Manager"
	Binary         InstallMethod = "Binary"
	FromSource     InstallMethod = "Source"
)

func main() {
	if os.Geteuid() == 0 {
		fmt.Println("❌ Error: Please do not run as root.")
		fmt.Println("   The installer will ask for sudo when needed.")
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("   ╔════════════════════════════════════╗")
	fmt.Println("   ║   🪷  fcitx5-lotus Installer  🪷   ║")
	fmt.Println("   ╚════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("💻 OS: %s  |  🏗️  Arch: %s\n\n", runtime.GOOS, runtime.GOARCH)

	// Step 1: Detect OS
	fmt.Println("🔍 Detecting system...")
	d, err := distro.Detect()
	if err != nil {
		fmt.Printf("❌ Failed to detect OS: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Detected: %s %s\n\n", d.Name, d.Version)

	if d.Type == distro.Unknown {
		fmt.Println("⚠️  Unsupported or unrecognized distribution.")
		fmt.Println("   Please install manually following the guide at:")
		fmt.Println("   https://lotusinputmethod.github.io/")
		os.Exit(1)
	}

	if d.Type == distro.NixOS {
		fmt.Println("🐧 NixOS detected!")
		fmt.Println("   Please use the flake/module method:")
		fmt.Println()
		fmt.Println("   Add to flake.nix:")
		fmt.Println("   ┌─────────────────────────────────────")
		fmt.Println("   │ inputs.fcitx5-lotus = {")
		fmt.Println("   │   url = \"github:LotusInputMethod/fcitx5-lotus\";")
		fmt.Println("   │   inputs.nixpkgs.follows = \"nixpkgs\";")
		fmt.Println("   │ };")
		fmt.Println("   └─────────────────────────────────────")
		fmt.Println()
		fmt.Println("   Add to configuration.nix:")
		fmt.Println("   ┌─────────────────────────────────────")
		fmt.Println("   │ services.fcitx5-lotus = {")
		fmt.Println("   │   enable = true;")
		fmt.Println("   │   user = \"your_username\";")
		fmt.Println("   │ };")
		fmt.Println("   └─────────────────────────────────────")
		fmt.Println()
		fmt.Println("   Then rebuild: sudo nixos-rebuild switch")
		os.Exit(0)
	}

	// Step 2: Detect init system
	initSys := detectInitSystem()
	fmt.Printf("⚙️  Init system: %s\n", initSys)

	// Step 3: Detect shell
	shell := detectShell()
	fmt.Printf("🐚 Shell: %s\n", shell)

	// Step 4: Select DE
	de := selectDE(reader)
	fmt.Printf("🖥️  Desktop: %s\n", de)

	// Step 5: Detect session
	session := detectSession()
	fmt.Printf("🪟 Session: %s\n\n", session)

	// Step 6: Select install method
	method := selectMethod(reader)
	fmt.Printf("📦 Method: %s\n\n", method)

	// Step 7: Install
	if method == PackageManager || method == Binary {
		fmt.Println("📦 Installing via package manager...")
		fmt.Println()
		fmt.Println("   Please run the following commands manually:")
		fmt.Println()

		switch d.Type {
		case distro.Arch:
			if method == PackageManager {
				fmt.Println("   yay -S fcitx5-lotus-bin")
			} else {
				fmt.Println("   Arch recommends using AUR for binary installation")
			}
		case distro.Debian, distro.Ubuntu:
			if method == PackageManager {
				fmt.Println("   curl -fsSL https://fcitx5-lotus.pages.dev/pubkey.gpg | sudo gpg --dearmor -o /etc/apt/keyrings/fcitx5-lotus.gpg")
				fmt.Printf("   echo \"deb [signed-by=/etc/apt/keyrings/fcitx5-lotus.gpg] https://fcitx5-lotus.pages.dev/apt/%s %s main\" | sudo tee /etc/apt/sources.list.d/fcitx5-lotus.list\n", d.Version, d.Version)
				fmt.Println("   sudo apt update && sudo apt install fcitx5-lotus")
			} else {
				fmt.Println("   sudo dpkg -i fcitx5-lotus_*.deb")
			}
		case distro.Fedora:
			if method == PackageManager {
				fmt.Printf("   sudo dnf config-manager addrepo --from-repofile=https://fcitx5-lotus.pages.dev/rpm/fedora/fcitx5-lotus-%s.repo\n", d.Version)
				fmt.Println("   sudo dnf install fcitx5-lotus")
			} else {
				fmt.Println("   sudo rpm -i fcitx5-lotus-*.rpm")
			}
		case distro.OpenSUSE:
			if method == PackageManager {
				fmt.Println("   sudo zypper addrepo https://fcitx5-lotus.pages.dev/rpm/opensuse/fcitx5-lotus-tumbleweed.repo")
				fmt.Println("   sudo zypper refresh")
				fmt.Println("   sudo zypper install fcitx5-lotus")
			} else {
				fmt.Println("   sudo rpm -i fcitx5-lotus-*.rpm")
			}
		case distro.VoidLinux:
			fmt.Println("   Void Linux does not have official packages yet.")
			fmt.Println("   Please build from source.")
		}

		fmt.Println()
		fmt.Println("   After installing, run the post-install configuration:")
		fmt.Println("   📦 fcitx5-lotus-installer --configure")
		os.Exit(0)
	}

	// FromSource flow
	fmt.Println("📦 Build from source selected\n")

	// Step 8: Check & install deps
	fmt.Println("📦 Checking dependencies...")
	allDeps := packages.AllDeps(d.Type)
	var missing []string
	for _, pkg := range allDeps {
		if !packages.IsPackageInstalled(pkg, d.Type) {
			missing = append(missing, pkg)
		}
	}

	if len(missing) > 0 {
		fmt.Printf("⚠️  Missing %d packages\n", len(missing))
		fmt.Println("📥 Installing dependencies...")
		if err := packages.InstallPackages(missing, d); err != nil {
			fmt.Printf("❌ Failed to install dependencies: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Dependencies installed\n")
	} else {
		fmt.Println("✅ All dependencies satisfied\n")
	}

	// Step 9: Clone
	home, _ := os.UserHomeDir()
	workDir := filepath.Join(home, ".cache", "fcitx5-lotus-installer")
	os.MkdirAll(workDir, 0755)

	b := build.NewBuilder(workDir)

	fmt.Println("📥 Cloning fcitx5-lotus...")
	if err := b.Clone(); err != nil {
		fmt.Printf("❌ Clone failed: %v\n", err)
		os.Exit(1)
	}

	// Step 10: Configure & Build
	fmt.Println("🔨 Configuring...")
	if err := b.Configure(); err != nil {
		fmt.Printf("❌ Configure failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🔨 Building...")
	if err := b.Build(); err != nil {
		fmt.Printf("❌ Build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("📤 Installing to system...")
	if err := b.Install(); err != nil {
		fmt.Printf("❌ Install failed: %v\n", err)
		os.Exit(1)
	}

	// Step 11: Post-install services
	fmt.Println()
	fmt.Println("🔧 Post-install configuration...")

	sm := services.New(
		services.InitSystem(initSys),
		string(configure.DesktopEnv(de)),
		string(configure.SessionEnv(session)),
	)

	fmt.Println("[1/4] Creating uinput_proxy user...")
	if err := sm.CreateUserAndGroup(); err != nil {
		fmt.Printf("⚠️  User creation warning: %v\n", err)
	}

	fmt.Println("[2/4] Reloading udev rules...")
	if err := sm.ReloadUdev(); err != nil {
		fmt.Printf("⚠️  Udev reload warning: %v\n", err)
	}

	fmt.Println("[3/4] Loading uinput module...")
	if err := sm.ModprobeUinput(); err != nil {
		fmt.Printf("⚠️  Uinput modprobe warning: %v\n", err)
	}

	fmt.Println("[4/4] Activating fcitx5-lotus-server...")
	if err := sm.ActivateServer(); err != nil {
		fmt.Printf("⚠️  Server activation warning: %v\n", err)
	}

	// Step 12: Kill IBus
	fmt.Println()
	fmt.Println("🔄 Checking for IBus...")
	sm.KillIBus()

	// Step 13: Configure environment
	fmt.Println()
	fmt.Println("⚙️  Setting up environment...")

	shellType := configure.ShellType(shell)
	deType := configure.DesktopEnv(de)
	sessionType := configure.SessionEnv(session)

	cfg, err := configure.NewConfigurer(shellType, deType, sessionType)
	if err != nil {
		fmt.Printf("❌ Config failed: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.ApplyAll(); err != nil {
		fmt.Printf("⚠️  Configuration warning: %v\n", err)
	}

	// Step 14: Restart fcitx5
	if cfg.CheckFcitx5Running() {
		fmt.Println("🔄 Restarting fcitx5...")
		if err := cfg.RestartFcitx5(); err != nil {
			fmt.Printf("⚠️  Could not restart fcitx5: %v\n", err)
			fmt.Println("   Manual: fcitx5 -d --replace")
		}
	}

	// Summary
	fmt.Println()
	fmt.Println("   ╔════════════════════════════════════╗")
	fmt.Println("   ║   ✅ Installation Complete!  ✅    ║")
	fmt.Println("   ╚════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("📝 Next steps:")
	fmt.Println("  1. Log out and log back in (or restart)")
	fmt.Println("  2. Open fcitx5-configtool")
	fmt.Println("  3. Add 'Lotus' input method to the left column")
	fmt.Println("  4. Start typing tiếng Việt!")

	// DE-specific tips
	if session == "Wayland" {
		fmt.Println()
		fmt.Println("🪟 Wayland notes:")
		if de == "KDE Plasma" {
			fmt.Println("  • Go to System Settings → Keyboard → Virtual Keyboard → Select Fcitx 5")
		}
		fmt.Println("  • Chromium/Electron flags:")
		fmt.Printf("    %s\n", sm.ChromiumWaylandFlags())
	}

	if initSys != "systemd" {
		fmt.Println()
		fmt.Println("⚠️  Non-systemd notes:")
		if session == "Wayland" {
			fmt.Println("  • Add to /etc/environment or DE config:")
			fmt.Println("    DBUS_SESSION_BUS_ADDRESS=unix:path=$XDG_RUNTIME_DIR/bus")
		}
	}

	fmt.Println()
	fmt.Println("🔧 Troubleshooting:")
	fmt.Println("  • Restart fcitx5: fcitx5 -d --replace")
	fmt.Println("  • Check status: fcitx5-diagnose")
	fmt.Println("  • Docs: https://lotusinputmethod.github.io/")

	b.Cleanup()
}

func detectInitSystem() string {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return "systemd"
	}
	if _, err := os.Stat("/etc/init.d"); err == nil {
		if _, err2 := os.Stat("/etc/rc.conf"); err2 == nil {
			return "openrc"
		}
	}
	if _, err := os.Stat("/etc/sv"); err == nil {
		return "runit"
	}
	if _, err := os.Stat("/etc/runit/sv"); err == nil {
		return "runit"
	}
	return "systemd"
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "bash"
	}
	if strings.Contains(shell, "zsh") {
		return "zsh"
	}
	if strings.Contains(shell, "fish") {
		return "fish"
	}
	return "bash"
}

func selectDE(reader *bufio.Reader) string {
	des := []string{
		"GNOME", "KDE Plasma", "Xfce", "Cinnamon", "MATE",
		"Pantheon", "Budgie", "LXQt", "COSMIC", "i3", "Sway", "Hyprland",
	}

	fmt.Println("🖥️  Select your Desktop Environment / WM:")
	for i, de := range des {
		fmt.Printf("  %d. %s\n", i+1, de)
	}
	fmt.Print("\nChoice [1]: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return "GNOME"
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(des) {
		return "GNOME"
	}
	return des[idx-1]
}

func detectSession() string {
	xdgSession := os.Getenv("XDG_SESSION_TYPE")
	if xdgSession == "wayland" {
		return "Wayland"
	}
	if xdgSession == "x11" {
		return "X11"
	}

	// Fallback checks
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "Wayland"
	}
	if os.Getenv("DISPLAY") != "" {
		return "X11"
	}

	return "X11"
}

func selectMethod(reader *bufio.Reader) InstallMethod {
	fmt.Println()
	fmt.Println("📦 Select install method:")
	fmt.Println("  1. Package Manager (recommended)")
	fmt.Println("  2. Build from Source")
	fmt.Print("\nChoice [1]: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "2":
		return FromSource
	default:
		return PackageManager
	}
}
