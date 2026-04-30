package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/hthienloc/fcitx5-lotus-installer/internal/build"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/configure"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/distro"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/packages"
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

func banner() {
	fmt.Println()
	fmt.Println(bold + magenta + "  🪷  fcitx5-lotus Installer" + reset)
	fmt.Println(dim + "  ──────────────────────────" + reset)
	fmt.Println()
}

func infoLine(label, value string) {
	fmt.Println("  " + dim + label + reset + " " + value)
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
	fmt.Printf("\n  %s", bold+label+reset)
	if def != "" {
		fmt.Printf(" [%s]", dim+def+reset)
	}
	fmt.Print(": ")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" && def != "" {
			return def
		}
		return input
	}
	return ""
}

func confirm(label string) bool {
	ans := prompt(label, "Y/n")
	if ans == "" || ans == "Y/n" {
		return true
	}
	l := strings.ToLower(ans)
	return l == "y" || l == "yes"
}

func pause() {
	fmt.Print("\n  " + dim + "Press Enter to continue" + reset)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
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

type installStatus struct {
	installed bool
	method    string
	version   string
}

func detectInstall(d distro.DistroInfo) installStatus {
	var out []byte
	var err error

	switch d.Type {
	case distro.Fedora, distro.OpenSUSE:
		out, err = exec.Command("rpm", "-q", "--qf", "%{VERSION}-%{RELEASE}", "fcitx5-lotus").CombinedOutput()
		if err == nil {
			return installStatus{installed: true, method: "rpm", version: strings.TrimSpace(string(out))}
		}
	case distro.Debian, distro.Ubuntu:
		out, err = exec.Command("dpkg", "-s", "fcitx5-lotus").CombinedOutput()
		if err == nil {
			return installStatus{installed: true, method: "deb"}
		}
	case distro.Arch:
		out, err = exec.Command("pacman", "-Q", "fcitx5-lotus").CombinedOutput()
		if err != nil {
			out, err = exec.Command("pacman", "-Q", "fcitx5-lotus-bin").CombinedOutput()
		}
		if err != nil {
			out, err = exec.Command("pacman", "-Q", "fcitx5-lotus-git").CombinedOutput()
		}
		if err == nil {
			return installStatus{installed: true, method: "aur", version: strings.TrimSpace(string(out))}
		}
	}

	_, err = exec.LookPath("fcitx5-lotus")
	if err == nil {
		out, _ = exec.Command("fcitx5-lotus", "--version").CombinedOutput()
		if len(out) > 0 {
			return installStatus{installed: true, method: "source", version: strings.TrimSpace(string(out))}
		}
		return installStatus{installed: true, method: "source"}
	}

	return installStatus{installed: false}
}

func uninstall(d distro.DistroInfo, method string) error {
	fmt.Println("  Removing fcitx5-lotus...")

	switch d.Type {
	case distro.Fedora, distro.OpenSUSE:
		cmds := [][]string{
			{"sudo", "dnf", "remove", "-y", "fcitx5-lotus"},
			{"sudo", "rm", "-f", "/etc/yum.repos.d/fcitx5-lotus-*.repo"},
		}
		for _, args := range cmds {
			out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("uninstall failed: %s\n%s", err, string(out))
			}
		}
	case distro.Debian, distro.Ubuntu:
		cmds := [][]string{
			{"sudo", "apt-get", "remove", "-y", "fcitx5-lotus"},
			{"sudo", "rm", "-f", "/etc/apt/sources.list.d/fcitx5-lotus.list"},
			{"sudo", "rm", "-f", "/etc/apt/keyrings/fcitx5-lotus.gpg"},
		}
		for _, args := range cmds {
			out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("uninstall failed: %s\n%s", err, string(out))
			}
		}
	case distro.Arch:
		helpers := [][]string{
			{"yay", "-Rns", "--noconfirm", "fcitx5-lotus", "fcitx5-lotus-bin", "fcitx5-lotus-git"},
			{"paru", "-Rns", "--noconfirm", "fcitx5-lotus", "fcitx5-lotus-bin", "fcitx5-lotus-git"},
		}
		for _, args := range helpers {
			if _, err := exec.LookPath(args[0]); err == nil {
				out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
				if err != nil {
					return fmt.Errorf("uninstall failed: %s\n%s", err, string(out))
				}
				break
			}
		}
	default:
		home, _ := os.UserHomeDir()
		srcDir := filepath.Join(home, ".cache", "fcitx5-lotus-installer", "fcitx5-lotus")
		if _, err := os.Stat(srcDir); err == nil {
			_, err := exec.Command("sudo", "make", "-C", srcDir, "uninstall").CombinedOutput()
			if err != nil {
				_ = exec.Command("sudo", "rm", "-f", "/usr/local/bin/fcitx5-lotus").Run()
				_ = exec.Command("sudo", "rm", "-rf", "/usr/local/share/fcitx5-lotus").Run()
			} else {
				_ = exec.Command("sudo", "rm", "-rf", filepath.Join(home, ".cache", "fcitx5-lotus-installer")).Run()
			}
		} else {
			if err := exec.Command("sudo", "rm", "-f", "/usr/local/bin/fcitx5-lotus").Run(); err != nil {
				return fmt.Errorf("uninstall failed: %s", err)
			}
		}
	}

	return nil
}

func arch() string {
	a := runtime.GOARCH
	if a == "amd64" {
		return "x86_64"
	}
	if a == "arm64" {
		return "aarch64"
	}
	return a
}

func ctx(d distro.DistroInfo, initSys, shell, session, de, step string) map[string]string {
	return map[string]string{
		"os":      d.Name + " " + d.Version,
		"arch":    arch(),
		"init":    initSys,
		"shell":   shell,
		"session": session,
		"de":      de,
		"step":    step,
	}
}

func die(msg string, err error, c map[string]string) {
	fmt.Println()
	fmt.Println(bold + red + "  ✗  " + msg + reset)
	if err != nil {
		fmt.Println("  " + red + err.Error() + reset)
	}
	fmt.Println()
	fmt.Println(dim + "  ── Debug Info ──" + reset)
	fmt.Printf(dim+"    OS:       %s\n"+reset, c["os"])
	fmt.Printf(dim+"    Arch:     %s\n"+reset, c["arch"])
	fmt.Printf(dim+"    Init:     %s\n"+reset, c["init"])
	fmt.Printf(dim+"    Shell:    %s\n"+reset, c["shell"])
	fmt.Printf(dim+"    Session:  %s\n"+reset, c["session"])
	fmt.Printf(dim+"    DE:       %s\n"+reset, c["de"])
	fmt.Printf(dim+"    Step:     %s\n"+reset, c["step"])
	if det, ok := c["detail"]; ok {
		fmt.Printf(dim+"    Detail:   %s\n"+reset, det)
	}
	fmt.Println()
	fmt.Println(dim + "  Report: https://github.com/hthienloc/fcitx5-lotus-installer/issues" + reset)
	fmt.Println()
	os.Exit(1)
}

func main() {
	if os.Geteuid() == 0 {
		fmt.Println(red + bold + "  Error:" + reset + red + " Do not run as root." + reset)
		os.Exit(1)
	}

	d, err := distro.Detect()
	if err != nil {
		die("Cannot detect OS", err, map[string]string{"step": "detection"})
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
		die("Unsupported distro", nil, map[string]string{"step": "detection"})
	}

	initSys := detectInitSystem()
	shell := detectShell()
	session := detectSession()
	de := ""

	fmt.Println(bold + cyan + "  System Detected" + reset)
	infoLine("OS:", d.Name+" "+d.Version)
	infoLine("Init:", initSys)
	infoLine("Shell:", shell)
	infoLine("Session:", session)
	fmt.Println()

	if !confirm("Continue with these settings") {
		fmt.Println("\n  " + dim + "Aborted." + reset)
		os.Exit(0)
	}

	status := detectInstall(d)

	if status.installed {
		fmt.Println()
		info("fcitx5-lotus is already installed")
		infoLine("Method:", status.method)
		if status.version != "" {
			infoLine("Version:", status.version)
		}
		fmt.Println()
		fmt.Println("  " + dim + "What would you like to do?" + reset)
		fmt.Println()
		fmt.Println("  1. Troubleshoot (repair/restart services)")
		fmt.Println("  2. Uninstall fcitx5-lotus")
		fmt.Println("  3. Reinstall / Update")
		fmt.Println("  4. Exit")
		fmt.Println()
		choice := prompt("Select action", "1")
		switch choice {
		case "2":
			fmt.Println()
			if !confirm("Remove fcitx5-lotus") {
				fmt.Println("\n  " + dim + "Aborted." + reset)
				os.Exit(0)
			}
			fmt.Println()
			if err := uninstall(d, status.method); err != nil {
				fail(err.Error())
				c := ctx(d, initSys, shell, session, "", "uninstall")
				die("Uninstall failed", err, c)
			}
			ok("fcitx5-lotus removed.")
			fmt.Println("\n  " + dim + "Done." + reset)
			return
		case "3":
			info("Proceeding with reinstall...")
		case "4":
			fmt.Println("\n  " + dim + "Done." + reset)
			return
		default:
			sm := services.New(services.InitSystem(initSys), "", session)
			fmt.Println()
			info("Restarting fcitx5-lotus services...")
			fmt.Println()
			if err := sm.Reload(); err != nil {
				fail(err.Error())
				fmt.Println()
				if confirm("Try full reinstall") {
					info("Proceeding with reinstall...")
				} else {
					fmt.Println("\n  " + dim + "Done." + reset)
					return
				}
			} else {
				ok("Services restarted.")
				fmt.Println("\n  " + dim + "Done." + reset)
				return
			}
		}
	}

	c := ctx(d, initSys, shell, session, "", "start")

	hasRepo := repo.HasOfficialRepo(d.Type)
	installMethod := "package"

	step(1, "Install fcitx5-lotus")

	c["step"] = "install"

	if hasRepo {
		info("Official repository available for " + d.Name)
		fmt.Println()
		if confirm("Install via package manager") {
			fmt.Println()
			if err := repo.SetupAndInstall(d); err != nil {
				fail(err.Error())
				fmt.Println()
				warn("Package install failed, falling back to source build...")
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
				warn("AUR install failed, falling back to source build...")
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
		if confirm("\n  Install build dependencies") {
			deps := packages.AllDeps(d.Type)
			if err := packages.InstallPackages(deps, d); err != nil {
				warn("Failed to install some dependencies. Continuing anyway...")
			} else {
				ok("Build dependencies installed.")
			}
		}

		if !confirm("\n  Clone and build fcitx5-lotus") {
			fmt.Println("\n  " + dim + "Aborted." + reset)
			os.Exit(0)
		}
		fmt.Println()

		home, _ := os.UserHomeDir()
		workDir := filepath.Join(home, ".cache", "fcitx5-lotus-installer")
		os.MkdirAll(workDir, 0755)
		b := build.NewBuilder(workDir)

		c["step"] = "build-from-source"

		fmt.Print("  " + dim + "Cloning repository... " + reset)
		if err := b.Clone(); err != nil {
			fmt.Println(red + "failed" + reset)
			die("Clone failed", err, c)
		}
		fmt.Println(green + "done" + reset)

		fmt.Print("  " + dim + "Configuring cmake... " + reset)
		if err := b.Configure(); err != nil {
			fmt.Println(red + "failed" + reset)
			die("CMake configure failed", err, c)
		}
		fmt.Println(green + "done" + reset)

		fmt.Print("  " + dim + "Building... " + reset)
		if err := b.Build(); err != nil {
			fmt.Println(red + "failed" + reset)
			die("Build failed", err, c)
		}
		fmt.Println(green + "done" + reset)

		fmt.Print("  " + dim + "Installing to system... " + reset)
		if err := b.Install(); err != nil {
			fmt.Println(red + "failed" + reset)
			die("Install failed", err, c)
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
	de = des[deIdx-1]
	c["de"] = de
	c["step"] = "post-install"
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

		cfg, _ := configure.NewConfigurer(
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

	if session == "Wayland" {
		fmt.Println("  " + bold + "Wayland notes:" + reset)
		if de == "KDE Plasma" {
			fmt.Println("    System Settings → Keyboard → Virtual Keyboard → Fcitx 5")
		}
		fmt.Println()
	}

	fmt.Println("  " + bold + "Next steps:" + reset)
	fmt.Println("    1. Log out and log back in (or reboot)")
	fmt.Println("    2. Open fcitx5-configtool")
	fmt.Println("    3. Add 'Lotus' to the left column")
	fmt.Println("    4. Start typing tiếng Việt!")
	fmt.Println()
	fmt.Println("  Docs: " + bold + "https://lotusinputmethod.github.io/" + reset)
	fmt.Println()

	fmt.Println(yellow + "  Note: Some environment changes require a logout or reboot to take effect." + reset)
	if confirm("Reboot system now") {
		fmt.Println()
		fmt.Println("  " + dim + "Rebooting..." + reset)
		fmt.Println()
		cmd := exec.Command("sudo", "reboot")
		cmd.Run()
	} else {
		fmt.Println("\n  " + dim + "Please remember to log out or reboot later." + reset)
		fmt.Println("  " + bold + "Done." + reset)
	}
}
