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
	"github.com/hthienloc/fcitx5-lotus-installer/internal/repo"
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
	w := 42
	fmt.Println(dim + "  ┌" + strings.Repeat("─", w) + "┐" + reset)
	fmt.Printf(dim+"  │ "+reset+bold+cyan+"%s"+reset+dim+"\n", title)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fmt.Printf(dim + "  │ " + reset + dim + "%s" + reset + "\n", line)
	}
	fmt.Println(dim + "  └" + strings.Repeat("─", w) + "┘" + reset)
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

func info(msg string) {
	fmt.Println("  " + cyan + "•" + reset + "  " + msg)
}

func prompt(label, def string) string {
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

	// Silent detection
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

	hasRepo := repo.HasOfficialRepo(d.Type)

	content := fmt.Sprintf("  OS:      %s %s\n  Init:    %s\n  Shell:   %s\n  Session: %s",
		d.Name, d.Version, initSys, shell, session)
	box("System Detected", content)

	if !confirm("  Continue with these settings") {
		fmt.Println("\n  " + dim + "Aborted." + reset)
		os.Exit(0)
	}

	// Step 1: Install via package manager (preferred)
	step(1, "Install fcitx5-lotus")
	fmt.Println()

	installMethod := "package"

	if hasRepo {
		fmt.Println("  Official repository available for " + d.Name)
		fmt.Println()

		if confirm("  Install via package manager (recommended)") {
			if err := repo.SetupAndInstall(d); err != nil {
				fail("Package install failed: " + err.Error())
				fmt.Println()
				fmt.Println("  Falling back to source build...")
				installMethod = "source"
			} else {
				ok("Installed via package manager.")
			}
		} else {
			fmt.Println()
			if confirm("  Build from source instead") {
				installMethod = "source"
			} else {
				fmt.Println("\n  " + dim + "Aborted." + reset)
				os.Exit(0)
			}
		}
	} else if d.Type == distro.Arch {
		fmt.Println("  AUR package available: fcitx5-lotus-bin")
		fmt.Println()

		if confirm("  Install via AUR helper (yay/paru)") {
			if err := repo.SetupAndInstall(d); err != nil {
				fail("AUR install failed: " + err.Error())
				fmt.Println()
				fmt.Println("  Falling back to source build...")
				installMethod = "source"
			} else {
				ok("Installed via AUR.")
			}
		} else {
			fmt.Println()
			if confirm("  Build from source instead") {
				installMethod = "source"
			} else {
				fmt.Println("\n  " + dim + "Aborted." + reset)
				os.Exit(0)
			}
		}
	} else {
		fmt.Println("  No official package for " + d.Name)
		fmt.Println("  Will build from source.")
		installMethod = "source"
	}

	// Step 2: Source build (if chosen or fallback)
	if installMethod == "source" {
		step(2, "Build from Source")
		fmt.Println()

		if !confirm("  Clone and build fcitx5-lotus") {
			fmt.Println("\n  " + dim + "Aborted." + reset)
			os.Exit(0)
		}

		fmt.Println()

		home, _ := os.UserHomeDir()
		workDir := filepath.Join(home, ".cache", "fcitx5-lotus-installer")
		os.MkdirAll(workDir, 0755)

		b := build.NewBuilder(workDir)

		fmt.Print("  " + dim + "Cloning repository... " + reset)
		if err := b.Clone(); err != nil {
			fmt.Println(red + "fail" + reset)
			fail(err.Error())
			os.Exit(1)
		}
		fmt.Println(green + "done" + reset)

		fmt.Print("  " + dim + "Configuring cmake... " + reset)
		if err := b.Configure(); err != nil {
			fmt.Println(red + "fail" + reset)
			fail(err.Error())
			os.Exit(1)
		}
		fmt.Println(green + "done" + reset)

		fmt.Print("  " + dim + "Building... " + reset)
		if err := b.Build(); err != nil {
			fmt.Println(red + "fail" + reset)
			fail(err.Error())
			os.Exit(1)
		}
		fmt.Println(green + "done" + reset)

		fmt.Print("  " + dim + "Installing to system... " + reset)
		if err := b.Install(); err != nil {
			fmt.Println(red + "fail" + reset)
			fail(err.Error())
			os.Exit(1)
		}
		fmt.Println(green + "done" + reset)
	}

	waitForEnter("  Press Enter to continue")

	// Step 3: Select DE
	step(3, "Desktop Environment")
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

	waitForEnter("  Press Enter to continue")

	// Step 4: Post-install services
	step(4, "Post-install Setup")
	fmt.Println()
	info("Create uinput_proxy user/group")
	info("Reload udev rules")
	info("Load uinput kernel module")
	info("Activate fcitx5-lotus-server service")
	fmt.Println()

	sm := services.New(
		services.InitSystem(initSys),
		de,
		session,
	)

	if !confirm("  Run post-install setup") {
		fmt.Println("\n  " + dim + "Skipped." + reset)
	} else {
		fmt.Println()

		fmt.Print("  " + dim + "Creating uinput_proxy user... " + reset)
		if err := sm.CreateUserAndGroup(); err != nil {
			fmt.Println(yellow + "skip" + reset)
			warn(err.Error())
		} else {
			fmt.Println(green + "done" + reset)
		}

		fmt.Print("  " + dim + "Reloading udev rules... " + reset)
		if err := sm.ReloadUdev(); err != nil {
			fmt.Println(yellow + "skip" + reset)
			warn(err.Error())
		} else {
			fmt.Println(green + "done" + reset)
		}

		fmt.Print("  " + dim + "Loading uinput module... " + reset)
		if err := sm.ModprobeUinput(); err != nil {
			fmt.Println(yellow + "skip" + reset)
			warn(err.Error())
		} else {
			fmt.Println(green + "done" + reset)
		}

		fmt.Print("  " + dim + "Activating server... " + reset)
		if err := sm.ActivateServer(); err != nil {
			fmt.Println(yellow + "skip" + reset)
			warn(err.Error())
		} else {
			fmt.Println(green + "done" + reset)
		}

		fmt.Print("  " + dim + "Killing IBus (if running)... " + reset)
		sm.KillIBus()
		fmt.Println(green + "done" + reset)
	}

	waitForEnter("  Press Enter to continue")

	// Step 5: Environment
	step(5, "Configure Environment")
	fmt.Println()
	info("Shell profile (" + shell + ")")
	info("fcitx5 input method profile")
	info("Autostart for " + de)
	fmt.Println()

	if !confirm("  Apply configuration") {
		fmt.Println("\n  " + dim + "Skipped." + reset)
	} else {
		fmt.Println()

		cfg, err := configure.NewConfigurer(
			configure.ShellType(shell),
			configure.DesktopEnv(de),
			configure.SessionEnv(session),
		)
		if err != nil {
			fail(err.Error())
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
	}

	waitForEnter("  Press Enter to continue")

	// Step 6: Restart fcitx5
	step(6, "Restart Fcitx5")
	fmt.Println()

	cfg, _ := configure.NewConfigurer(
		configure.ShellType(shell),
		configure.DesktopEnv(de),
		configure.SessionEnv(session),
	)

	if cfg.CheckFcitx5Running() {
		if confirm("  Restart fcitx5 now") {
			if err := cfg.RestartFcitx5(); err != nil {
				warn("Run manually: fcitx5 -d --replace")
			} else {
				ok("Fcitx5 restarted.")
			}
		}
	} else {
		ok("Fcitx5 not running. Start with: fcitx5 -d")
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
		fmt.Println("    • Chromium: --enable-features=UseOzonePlatform --ozone-platform=wayland --enable-wayland-ime")
		fmt.Println()
	}

	if initSys != "systemd" && session == "Wayland" {
		fmt.Println("  " + bold + "Non-systemd note:" + reset)
		fmt.Println("    • Add DBUS_SESSION_BUS_ADDRESS=unix:path=$XDG_RUNTIME_DIR/bus")
		fmt.Println()
	}

	fmt.Println("  Docs: " + bold + "https://lotusinputmethod.github.io/" + reset)
	fmt.Println()
}
