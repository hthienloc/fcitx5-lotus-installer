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
	fmt.Println(bold + magenta + "  ╭──────────────────────────────────────────╮" + reset)
	fmt.Println(bold + magenta + "  │          🪷  fcitx5-lotus Installer        │" + reset)
	fmt.Println(bold + magenta + "  ╰──────────────────────────────────────────╯" + reset)
	fmt.Println()
}

func box(title string, lines []string) {
	maxLen := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if len(l) > maxLen {
			maxLen = len(l)
		}
	}
	if len(title) > maxLen {
		maxLen = len(title)
	}
	w := maxLen + 4

	bar := dim + "  +" + strings.Repeat("-", w) + "+" + reset
	fmt.Println(bar)
	fmt.Printf(dim+"  | "+reset+bold+cyan+"%-"+strconv.Itoa(maxLen)+"s"+reset+dim+" |\n", title)
	for _, l := range lines {
		fmt.Printf(dim+"  | "+reset+"%-"+strconv.Itoa(maxLen)+"s"+dim+" |\n", strings.TrimSpace(l))
	}
	fmt.Println(bar)
	fmt.Println()
}

func step(num int, title string) {
	fmt.Println()
	fmt.Println(bold + "  ── Step " + strconv.Itoa(num) + ": " + title + " ──" + reset)
}

func ok(msg string) {
	fmt.Println("  " + green + "✓" + reset + " " + msg)
}

func warn(msg string) {
	fmt.Println("  " + yellow + "!" + reset + " " + msg)
}

func fail(msg string) {
	fmt.Println("  " + red + "✗" + reset + " " + msg)
}

func info(msg string) {
	fmt.Println("  " + cyan + "•" + reset + " " + msg)
}

func prompt(label, def string) string {
	fmt.Print("\n  " + bold + label + reset)
	if def != "" {
		fmt.Printf(" [%s]", dim+def+reset)
	}
	fmt.Print(": ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" && def != "" {
		return def
	}
	return input
}

func confirm(label string) bool {
	ans := prompt(label, "Y/n")
	l := strings.ToLower(ans)
	return l == "y" || l == "yes" || ans == "" || ans == "Y"
}

func pause() {
	fmt.Print("\n  " + dim + "Press Enter to continue" + reset)
	reader.ReadString('\n')
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
		os.Exit(1)
	}

	banner()

	d, err := distro.Detect()
	if err != nil {
		fmt.Println(red + "  Cannot detect OS: " + err.Error() + reset)
		os.Exit(1)
	}

	if d.Type == distro.NixOS {
		fmt.Println("  " + cyan + "NixOS detected." + reset)
		fmt.Println()
		fmt.Println("  Please use the flake method.")
		fmt.Println("  " + bold + "https://lotusinputmethod.github.io/" + reset)
		fmt.Println()
		os.Exit(0)
	}

	if d.Type == distro.Unknown {
		fmt.Println(red + "  Unsupported distro." + reset)
		fmt.Println("  " + bold + "https://lotusinputmethod.github.io/" + reset)
		os.Exit(1)
	}

	initSys := detectInitSystem()
	shell := detectShell()
	session := detectSession()

	box("System Detected", []string{
		"OS:      " + d.Name + " " + d.Version,
		"Init:    " + initSys,
		"Shell:   " + shell,
		"Session: " + session,
	})

	if !confirm("Continue with these settings") {
		fmt.Println("\n  " + dim + "Aborted." + reset)
		os.Exit(0)
	}

	hasRepo := repo.HasOfficialRepo(d.Type)
	installMethod := "package"

	step(1, "Install fcitx5-lotus")

	if hasRepo {
		info("Official repository available for " + d.Name)
		fmt.Println()
		if confirm("Install via package manager") {
			fmt.Println()
			if err := repo.SetupAndInstall(d); err != nil {
				fail(err.Error())
				fmt.Println()
				info("Falling back to source build...")
				installMethod = "source"
			} else {
				ok("Installed via package manager.")
			}
		} else {
			fmt.Println()
			if confirm("Build from source instead") {
				installMethod = "source"
			} else {
				fmt.Println("\n  " + dim + "Aborted." + reset)
				os.Exit(0)
			}
		}
	} else if d.Type == distro.Arch {
		info("AUR package available: fcitx5-lotus-bin")
		fmt.Println()
		if confirm("Install via AUR helper") {
			fmt.Println()
			if err := repo.SetupAndInstall(d); err != nil {
				fail(err.Error())
				fmt.Println()
				info("Falling back to source build...")
				installMethod = "source"
			} else {
				ok("Installed via AUR.")
			}
		} else {
			fmt.Println()
			if confirm("Build from source instead") {
				installMethod = "source"
			} else {
				fmt.Println("\n  " + dim + "Aborted." + reset)
				os.Exit(0)
			}
		}
	} else {
		info("No official package for " + d.Name)
		info("Building from source...")
		installMethod = "source"
	}

	if installMethod == "source" {
		if !confirm("\n  Clone and build fcitx5-lotus") {
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
			fmt.Println(red + "failed" + reset)
			fail(err.Error())
			os.Exit(1)
		}
		fmt.Println(green + "done" + reset)

		fmt.Print("  " + dim + "Configuring cmake... " + reset)
		if err := b.Configure(); err != nil {
			fmt.Println(red + "failed" + reset)
			fail(err.Error())
			os.Exit(1)
		}
		fmt.Println(green + "done" + reset)

		fmt.Print("  " + dim + "Building... " + reset)
		if err := b.Build(); err != nil {
			fmt.Println(red + "failed" + reset)
			fail(err.Error())
			os.Exit(1)
		}
		fmt.Println(green + "done" + reset)

		fmt.Print("  " + dim + "Installing to system... " + reset)
		if err := b.Install(); err != nil {
			fmt.Println(red + "failed" + reset)
			fail(err.Error())
			os.Exit(1)
		}
		fmt.Println(green + "done" + reset)
	}

	pause()

	step(2, "Desktop Environment")

	des := []string{"GNOME", "KDE Plasma", "Xfce", "Cinnamon", "MATE", "Pantheon", "Budgie", "LXQt", "COSMIC", "i3", "Sway", "Hyprland"}
	fmt.Println()
	for i, de := range des {
		fmt.Printf("  %2d. %s\n", i+1, de)
	}
	deIdx, _ := strconv.Atoi(prompt("Select desktop", "1"))
	if deIdx < 1 || deIdx > len(des) {
		deIdx = 1
	}
	de := des[deIdx-1]
	ok("Selected: " + de)

	pause()

	step(3, "Post-install Setup")
	fmt.Println()
	info("Create uinput_proxy user/group")
	info("Reload udev rules")
	info("Load uinput kernel module")
	info("Activate fcitx5-lotus-server service")
	fmt.Println()

	sm := services.New(services.InitSystem(initSys), de, session)

	if !confirm("Run post-install setup") {
		fmt.Println("\n  " + dim + "Skipped." + reset)
	} else {
		fmt.Println()

		fmt.Print("  " + dim + "Creating uinput_proxy user... " + reset)
		if err := sm.CreateUserAndGroup(); err != nil {
			fmt.Println(yellow + "skip" + reset)
		} else {
			fmt.Println(green + "done" + reset)
		}

		fmt.Print("  " + dim + "Reloading udev rules... " + reset)
		if err := sm.ReloadUdev(); err != nil {
			fmt.Println(yellow + "skip" + reset)
		} else {
			fmt.Println(green + "done" + reset)
		}

		fmt.Print("  " + dim + "Loading uinput module... " + reset)
		if err := sm.ModprobeUinput(); err != nil {
			fmt.Println(yellow + "skip" + reset)
		} else {
			fmt.Println(green + "done" + reset)
		}

		fmt.Print("  " + dim + "Activating server... " + reset)
		if err := sm.ActivateServer(); err != nil {
			fmt.Println(yellow + "skip" + reset)
		} else {
			fmt.Println(green + "done" + reset)
		}

		fmt.Print("  " + dim + "Killing IBus... " + reset)
		sm.KillIBus()
		fmt.Println(green + "done" + reset)
	}

	pause()

	step(4, "Configure Environment")
	fmt.Println()
	info("Shell profile (" + shell + ")")
	info("fcitx5 input method profile")
	info("Autostart for " + de)
	fmt.Println()

	if !confirm("Apply configuration") {
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

	pause()

	step(5, "Restart Fcitx5")
	fmt.Println()

	cfg, _ := configure.NewConfigurer(
		configure.ShellType(shell),
		configure.DesktopEnv(de),
		configure.SessionEnv(session),
	)

	if cfg.CheckFcitx5Running() {
		if confirm("Restart fcitx5 now") {
			if err := cfg.RestartFcitx5(); err != nil {
				warn("Run: fcitx5 -d --replace")
			} else {
				ok("Fcitx5 restarted.")
			}
		}
	} else {
		ok("Fcitx5 not running. Start with: fcitx5 -d")
	}

	fmt.Println()
	fmt.Println(bold + magenta + "  ╭──────────────────────────────────────────╮" + reset)
	fmt.Println(bold + magenta + "  │           ✅  Installation Done!          │" + reset)
	fmt.Println(bold + magenta + "  ╰──────────────────────────────────────────╯" + reset)
	fmt.Println()
	fmt.Println("  " + bold + "Next steps:" + reset)
	fmt.Println("    1. Log out and log back in")
	fmt.Println("    2. Open fcitx5-configtool")
	fmt.Println("    3. Add 'Lotus' to the left column")
	fmt.Println("    4. Start typing tiếng Việt!")
	fmt.Println()

	if session == "Wayland" {
		fmt.Println("  " + bold + "Wayland notes:" + reset)
		if de == "KDE Plasma" {
			fmt.Println("    System Settings → Keyboard → Virtual Keyboard → Fcitx 5")
		}
		fmt.Println("    Chromium: --enable-features=UseOzonePlatform --ozone-platform=wayland")
		fmt.Println()
	}

	fmt.Println("  Docs: " + bold + "https://lotusinputmethod.github.io/" + reset)
	fmt.Println()
}
