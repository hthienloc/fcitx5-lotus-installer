package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hthienloc/fcitx5-lotus-installer/internal/build"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/configure"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/distro"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/packages"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/services"
)

const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	red     = "\033[31m"
	bright  = "\033[1;37m"
)

var reader = bufio.NewReader(os.Stdin)

func banner() {
	fmt.Println()
	fmt.Println(bold + magenta + "  ╭─────────────────────────────────────────╮" + reset)
	fmt.Println(bold + magenta + "  │         🪷  fcitx5-lotus Installer       │" + reset)
	fmt.Println(bold + magenta + "  ╰─────────────────────────────────────────╯" + reset)
	fmt.Println()
}

func box(title, content string) {
	fmt.Println(dim + "  ┌" + strings.Repeat("─", 40) + "┐" + reset)
	fmt.Println(dim + "  │ " + reset + bold + cyan + title + reset + dim)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fmt.Printf(dim + "  │ " + reset + dim + "%s" + reset + "\n", line)
	}
	fmt.Println(dim + "  └" + strings.Repeat("─", 40) + "┘" + reset)
	fmt.Println()
}

func step(num int, title string) {
	fmt.Println()
	fmt.Println(bold + "  Step " + strconv.Itoa(num) + ": " + title + reset)
	fmt.Println(dim + "  " + strings.Repeat("─", 40) + reset)
}

func ok(msg string) {
	fmt.Println("  " + green + "✓" + reset + "  " + msg)
}

func warn(msg string) {
	fmt.Println("  " + yellow + "⚠" + reset + "  " + msg)
}

func fail(msg string) {
	fmt.Println("  " + red + "✗" + reset + "  " + msg)
}

func prompt(label string, def string) string {
	fmt.Printf("\n  %s", bold+label+reset)
	if def != "" {
		fmt.Printf(" [" + dim + "%s" + reset + "]", def)
	}
	fmt.Printf(": ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" && def != "" {
		return def
	}
	return input
}

func confirm(label string) bool {
	ans := prompt(label+"? (Y/n)", "Y")
	return strings.ToLower(ans) == "y" || strings.ToLower(ans) == "yes" || ans == "Y"
}

func waitForEnter(msg string) {
	prompt(msg, "")
}

func detectInitSystem() string {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return "systemd"
	}
	if _, err := os.Stat("/etc/sv"); err == nil {
		return "runit"
	}
	if _, err := os.Stat("/etc/runit/sv"); err == nil {
		return "runit"
	}
	if _, err := os.Stat("/etc/init.d"); err == nil {
		return "openrc"
	}
	return "systemd"
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return "zsh"
	}
	if strings.Contains(shell, "fish") {
		return "fish"
	}
	return "bash"
}

func detectSession() string {
	xdg := os.Getenv("XDG_SESSION_TYPE")
	if xdg == "wayland" {
		return "Wayland"
	}
	if xdg == "x11" {
		return "X11"
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "Wayland"
	}
	return "X11"
}

func main() {
	if os.Geteuid() == 0 {
		fmt.Println(red + bold + "  Error:" + reset + red + " Do not run as root." + reset)
		fmt.Println("  The installer will ask for sudo when needed.")
		os.Exit(1)
	}

	banner()

	// Step 0: Detect everything silently first
	d, err := distro.Detect()
	if err != nil {
		fmt.Println(red + "  Cannot detect OS: " + err.Error() + reset)
		os.Exit(1)
	}

	if d.Type == distro.NixOS {
		fmt.Println(cyan + "  NixOS detected." + reset)
		fmt.Println()
		fmt.Println("  This installer does not support NixOS.")
		fmt.Println("  Please use the flake method. See:")
		fmt.Println("  " + bold + "https://lotusinputmethod.github.io/" + reset)
		fmt.Println()
		os.Exit(0)
	}

	if d.Type == distro.Unknown {
		fmt.Println(red + "  Unsupported or unrecognized distro." + reset)
		fmt.Println("  Please install manually. See:")
		fmt.Println("  " + bold + "https://lotusinputmethod.github.io/" + reset)
		fmt.Println()
		os.Exit(1)
	}

	initSys := detectInitSystem()
	shell := detectShell()
	session := detectSession()

	// Show summary
	content := fmt.Sprintf("  OS:      %s %s\n  Init:    %s\n  Shell:   %s\n  Session: %s",
		d.Name, d.Version, initSys, shell, session)
	box("System Detected", content)

	if !confirm("  Continue with these settings") {
		fmt.Println("\n  " + dim + "Aborted." + reset)
		os.Exit(0)
	}

	// Step 1: Install method
	step(1, "Install Method")
	fmt.Println()
	fmt.Println("  1. Build from source (recommended for full setup)")
	fmt.Println("  2. Show package manager commands (manual install)")
	fmt.Println()

	method := prompt("  Choice", "1")

	if method == "2" {
		fmt.Println()
		fmt.Println(dim + "  ┌" + strings.Repeat("─", 50) + "┐" + reset)
		fmt.Println(dim + "  │" + reset + "  Run these commands manually:" + dim)
		fmt.Println(dim + "  │" + reset)

		switch d.Type {
		case distro.Arch:
			fmt.Println(dim + "  │" + reset + "    yay -S fcitx5-lotus-bin")
		case distro.Debian, distro.Ubuntu:
			fmt.Println(dim + "  │" + reset + "    curl -fsSL https://fcitx5-lotus.pages.dev/pubkey.gpg | \\")
			fmt.Println(dim + "  │" + reset + "      sudo gpg --dearmor -o /etc/apt/keyrings/fcitx5-lotus.gpg")
			fmt.Println(dim + "  │" + reset + "    echo \"deb [signed-by=...] https://fcitx5-lotus.pages.dev/apt/...\" | \\")
			fmt.Println(dim + "  │" + reset + "      sudo tee /etc/apt/sources.list.d/fcitx5-lotus.list")
			fmt.Println(dim + "  │" + reset + "    sudo apt update && sudo apt install fcitx5-lotus")
		case distro.Fedora:
			fmt.Println(dim + "  │" + reset + "    sudo dnf install fcitx5-lotus")
		case distro.OpenSUSE:
			fmt.Println(dim + "  │" + reset + "    sudo zypper install fcitx5-lotus")
		case distro.VoidLinux:
			fmt.Println(dim + "  │" + reset + "    Void Linux: build from source (no package yet)")
		}

		fmt.Println(dim + "  │" + reset)
		fmt.Println(dim + "  └" + strings.Repeat("─", 50) + "┘" + reset)
		fmt.Println()
		fmt.Println("  Full guide: " + bold + "https://lotusinputmethod.github.io/" + reset)
		fmt.Println()
		os.Exit(0)
	}

	// Step 2: Select DE
	step(2, "Desktop Environment")
	fmt.Println()
	des := []string{"GNOME", "KDE Plasma", "Xfce", "Cinnamon", "MATE", "Pantheon", "Budgie", "LXQt", "COSMIC", "i3", "Sway", "Hyprland"}
	for i, de := range des {
		fmt.Printf("  %2d. %s\n", i+1, de)
	}
	fmt.Println()

	deChoice := prompt("  Desktop", "1")
	deIdx, _ := strconv.Atoi(deChoice)
	if deIdx < 1 || deIdx > len(des) {
		deIdx = 1
	}
	de := des[deIdx-1]

	fmt.Println()
	ok("Selected: " + de)

	// Step 3: Dependencies
	step(3, "Check & Install Dependencies")
	fmt.Println()

	allDeps := packages.AllDeps(d.Type)
	var missing []string
	for _, pkg := range allDeps {
		if !packages.IsPackageInstalled(pkg, d.Type) {
			missing = append(missing, pkg)
		}
	}

	if len(missing) == 0 {
		ok("All dependencies satisfied.")
	} else {
		fmt.Println("  " + yellow + fmt.Sprintf("%d packages missing:", len(missing)) + reset)
		for _, pkg := range missing {
			fmt.Printf("    • %s\n", pkg)
		}
		fmt.Println()

		if confirm("  Install these packages") {
			if err := packages.InstallPackages(missing, d); err != nil {
				fail("Dependency install failed: " + err.Error())
				os.Exit(1)
			}
			ok("Dependencies installed.")
		} else {
			fmt.Println("\n  " + dim + "Aborted." + reset)
			os.Exit(0)
		}
	}

	waitForEnter("  Press Enter to continue")

	// Step 4: Clone
	step(4, "Clone Source")
	fmt.Println()

	if confirm("  Clone fcitx5-lotus repository") {
		home, _ := os.UserHomeDir()
		workDir := filepath.Join(home, ".cache", "fcitx5-lotus-installer")
		os.MkdirAll(workDir, 0755)

		b := build.NewBuilder(workDir)
		if err := b.Clone(); err != nil {
			fail("Clone failed: " + err.Error())
			os.Exit(1)
		}
		ok("Repository cloned.")
	} else {
		fmt.Println("\n  " + dim + "Aborted." + reset)
		os.Exit(0)
	}

	waitForEnter("  Press Enter to continue")

	// Step 5: Build
	step(5, "Build")
	fmt.Println()

	if confirm("  Run cmake and build") {
		home, _ := os.UserHomeDir()
		workDir := filepath.Join(home, ".cache", "fcitx5-lotus-installer")
		b := build.NewBuilder(workDir)

		if err := b.Configure(); err != nil {
			fail("Configure failed: " + err.Error())
			os.Exit(1)
		}
		if err := b.Build(); err != nil {
			fail("Build failed: " + err.Error())
			os.Exit(1)
		}
		ok("Build complete.")
	} else {
		fmt.Println("\n  " + dim + "Aborted." + reset)
		os.Exit(0)
	}

	waitForEnter("  Press Enter to continue")

	// Step 6: Install to system
	step(6, "Install to System")
	fmt.Println()

	if confirm("  Run sudo make install") {
		home, _ := os.UserHomeDir()
		workDir := filepath.Join(home, ".cache", "fcitx5-lotus-installer")
		b := build.NewBuilder(workDir)

		if err := b.Install(); err != nil {
			fail("Install failed: " + err.Error())
			os.Exit(1)
		}
		ok("Installed to /usr.")
	} else {
		fmt.Println("\n  " + dim + "Aborted." + reset)
		os.Exit(0)
	}

	waitForEnter("  Press Enter to continue")

	// Step 7: Post-install services
	step(7, "Post-install Setup")
	fmt.Println()
	fmt.Println("  The following will be done:")
	fmt.Println("    • Create uinput_proxy user/group")
	fmt.Println("    • Reload udev rules")
	fmt.Println("    • Load uinput kernel module")
	fmt.Println("    • Activate fcitx5-lotus-server service")
	fmt.Println()

	sm := services.New(
		services.InitSystem(initSys),
		de,
		session,
	)

	if confirm("  Run post-install setup") {
		fmt.Println()

		fmt.Print("  " + dim + "Creating uinput_proxy user... " + reset)
		if err := sm.CreateUserAndGroup(); err != nil {
			fmt.Println(yellow + "skip" + reset)
			warn("User creation: " + err.Error())
		} else {
			fmt.Println(green + "done" + reset)
		}

		fmt.Print("  " + dim + "Reloading udev rules... " + reset)
		if err := sm.ReloadUdev(); err != nil {
			fmt.Println(yellow + "skip" + reset)
			warn("Udev reload: " + err.Error())
		} else {
			fmt.Println(green + "done" + reset)
		}

		fmt.Print("  " + dim + "Loading uinput module... " + reset)
		if err := sm.ModprobeUinput(); err != nil {
			fmt.Println(yellow + "skip" + reset)
			warn("Uinput modprobe: " + err.Error())
		} else {
			fmt.Println(green + "done" + reset)
		}

		fmt.Print("  " + dim + "Activating server... " + reset)
		if err := sm.ActivateServer(); err != nil {
			fmt.Println(yellow + "skip" + reset)
			warn("Server activation: " + err.Error())
		} else {
			fmt.Println(green + "done" + reset)
		}

		fmt.Print("  " + dim + "Killing IBus (if running)... " + reset)
		sm.KillIBus()
		fmt.Println(green + "done" + reset)
	} else {
		fmt.Println("\n  " + dim + "Skipped. You can run these manually later." + reset)
	}

	waitForEnter("  Press Enter to continue")

	// Step 8: Environment
	step(8, "Configure Environment")
	fmt.Println()
	fmt.Println("  The following will be configured:")
	fmt.Println("    • Shell profile (" + shell + ")")
	fmt.Println("    • fcitx5 input method profile")
	fmt.Println("    • Autostart for " + de)
	fmt.Println()

	if confirm("  Apply configuration") {
		fmt.Println()

		cfg, err := configure.NewConfigurer(
			configure.ShellType(shell),
			configure.DesktopEnv(de),
			configure.SessionEnv(session),
		)
		if err != nil {
			fail("Config init failed: " + err.Error())
		} else {
			fmt.Print("  " + dim + "Setting up environment.d... " + reset)
			if err := cfg.SetupEnvironmentD(); err != nil {
				fmt.Println(yellow + "skip" + reset)
			} else {
				fmt.Println(green + "done" + reset)
			}

			fmt.Print("  " + dim + "Writing shell profile... " + reset)
			if err := cfg.SetupShellProfile(); err != nil {
				fmt.Println(yellow + "skip" + reset)
			} else {
				fmt.Println(green + "done" + reset)
			}

			fmt.Print("  " + dim + "Creating fcitx5 profile... " + reset)
			if err := cfg.SetupFcitx5Profile(); err != nil {
				fmt.Println(yellow + "skip" + reset)
			} else {
				fmt.Println(green + "done" + reset)
			}

			fmt.Print("  " + dim + "Setting up autostart... " + reset)
			if err := cfg.SetupAutostart(); err != nil {
				fmt.Println(yellow + "skip" + reset)
			} else {
				fmt.Println(green + "done" + reset)
			}
		}
	} else {
		fmt.Println("\n  " + dim + "Skipped. You can configure manually." + reset)
	}

	waitForEnter("  Press Enter to continue")

	// Step 9: Restart fcitx5
	step(9, "Restart Fcitx5")
	fmt.Println()

	cfg, _ := configure.NewConfigurer(
		configure.ShellType(shell),
		configure.DesktopEnv(de),
		configure.SessionEnv(session),
	)

	if cfg.CheckFcitx5Running() {
		if confirm("  Restart fcitx5 now") {
			if err := cfg.RestartFcitx5(); err != nil {
				warn("Restart failed. Run manually: fcitx5 -d --replace")
			} else {
				ok("Fcitx5 restarted.")
			}
		}
	} else {
		ok("Fcitx5 is not running. Start it with: fcitx5 -d")
	}

	// Done
	fmt.Println()
	fmt.Println(bold + magenta + "  ╭─────────────────────────────────────────╮" + reset)
	fmt.Println(bold + magenta + "  │           ✅ Installation Done!          │" + reset)
	fmt.Println(bold + magenta + "  ╰─────────────────────────────────────────╯" + reset)
	fmt.Println()
	fmt.Println("  " + bold + "Next steps:" + reset)
	fmt.Println("    1. Log out and log back in")
	fmt.Println("    2. Open fcitx5-configtool")
	fmt.Println("    3. Add 'Lotus' to the left column")
	fmt.Println("    4. Start typing tiếng Việt! 🪷")
	fmt.Println()

	if session == "Wayland" {
		fmt.Println("  " + bold + "Wayland notes:" + reset)
		if de == "KDE Plasma" {
			fmt.Println("    • System Settings → Keyboard → Virtual Keyboard → Fcitx 5")
		}
		fmt.Println("    • Chromium flags: --enable-features=UseOzonePlatform --ozone-platform=wayland --enable-wayland-ime")
		fmt.Println()
	}

	if initSys != "systemd" && session == "Wayland" {
		fmt.Println("  " + bold + "Non-systemd note:" + reset)
		fmt.Println("    • Add DBUS_SESSION_BUS_ADDRESS=unix:path=$XDG_RUNTIME_DIR/bus")
		fmt.Println("      to /etc/environment or your DE config")
		fmt.Println()
	}

	fmt.Println("  Docs: " + bold + "https://lotusinputmethod.github.io/" + reset)
	fmt.Println()
}
