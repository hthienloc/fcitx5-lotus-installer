package repo

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/hthienloc/fcitx5-lotus-installer/internal/distro"
)

const (
	pubKeyURL = "https://fcitx5-lotus.pages.dev/pubkey.gpg"
	baseURL   = "https://fcitx5-lotus.pages.dev"
)

func HasOfficialRepo(dt distro.DistroType) bool {
	switch dt {
	case distro.Debian, distro.Ubuntu, distro.Fedora, distro.OpenSUSE:
		return true
	default:
		return false
	}
}

func SetupAndInstall(d distro.DistroInfo) error {
	switch d.Type {
	case distro.Debian, distro.Ubuntu:
		return installApt(d)
	case distro.Fedora:
		return installDnf(d)
	case distro.OpenSUSE:
		return installZypper(d)
	case distro.Arch:
		return installAur()
	default:
		return fmt.Errorf("no official repo for %s", d.Type)
	}
}

func installApt(d distro.DistroInfo) error {
	var codename string
	if d.Type == distro.Ubuntu {
		out, _ := exec.Command("grep", "^UBUNTU_CODENAME=", "/etc/os-release").CombinedOutput()
		codename = strings.TrimPrefix(strings.TrimSpace(string(out)), "UBUNTU_CODENAME=")
	} else {
		out, _ := exec.Command("grep", "^VERSION_CODENAME=", "/etc/os-release").CombinedOutput()
		codename = strings.TrimPrefix(strings.TrimSpace(string(out)), "VERSION_CODENAME=")
	}

	if codename == "" {
		return fmt.Errorf("cannot detect codename")
	}

	fmt.Println("  Adding fcitx5-lotus repository...")

	cmds := [][]string{
		{"sudo", "mkdir", "-p", "/etc/apt/keyrings"},
		{"bash", "-c", fmt.Sprintf("curl -fsSL %s | sudo gpg --dearmor -o /etc/apt/keyrings/fcitx5-lotus.gpg", pubKeyURL)},
		{"bash", "-c", fmt.Sprintf("echo \"deb [signed-by=/etc/apt/keyrings/fcitx5-lotus.gpg] %s/apt/%s %s main\" | sudo tee /etc/apt/sources.list.d/fcitx5-lotus.list", baseURL, codename, codename)},
		{"sudo", "apt", "update"},
		{"sudo", "apt-get", "install", "-y", "fcitx5-lotus"},
	}

	for i, args := range cmds {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("step %d failed: %s\n%s", i+1, err, string(out))
		}
	}

	return nil
}

func installDnf(d distro.DistroInfo) error {
	fmt.Println("  Adding fcitx5-lotus repository...")

	repoFile := fmt.Sprintf("fcitx5-lotus-%s.repo", d.Version)
	repoURL := fmt.Sprintf("%s/rpm/fedora/%s", baseURL, repoFile)

	// Check if the versioned repo file exists and is not an HTML page
	checkCmd := fmt.Sprintf("curl -sSfL -I %s 2>/dev/null | grep -q 'Content-Type: text/plain' || curl -sSfL -I %s 2>/dev/null | grep -q 'application/octet-stream' || curl -sSfL -I %s 2>/dev/null | grep -v -q 'text/html'", repoURL, repoURL, repoURL)
	if err := exec.Command("bash", "-c", checkCmd).Run(); err != nil {
		fmt.Println("  Versioned repo not found, falling back to rawhide...")
		repoURL = fmt.Sprintf("%s/rpm/fedora/fcitx5-lotus-rawhide.repo", baseURL)
	}

	cmds := [][]string{
		{"bash", "-c", fmt.Sprintf("sudo rpm --import %s", pubKeyURL)},
		{"bash", "-c", fmt.Sprintf("sudo dnf config-manager addrepo --from-repofile=%s", repoURL)},
		{"sudo", "dnf", "install", "-y", "fcitx5-lotus"},
	}

	for i, args := range cmds {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("step %d failed: %s\n%s", i+1, err, string(out))
		}
	}

	return nil
}

func installZypper(d distro.DistroInfo) error {
	fmt.Println("  Adding fcitx5-lotus repository...")

	cmds := [][]string{
		{"bash", "-c", fmt.Sprintf("sudo rpm --import %s", pubKeyURL)},
		{"bash", "-c", fmt.Sprintf("sudo zypper addrepo %s/rpm/opensuse/fcitx5-lotus-tumbleweed.repo", baseURL)},
		{"sudo", "zypper", "refresh"},
		{"sudo", "zypper", "install", "-y", "fcitx5-lotus"},
	}

	for i, args := range cmds {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("step %d failed: %s\n%s", i+1, err, string(out))
		}
	}

	return nil
}

func installAur() error {
	fmt.Println("  Installing via AUR helper...")

	helpers := [][]string{
		{"yay", "-S", "--noconfirm", "fcitx5-lotus-bin"},
		{"paru", "-S", "--noconfirm", "fcitx5-lotus-bin"},
	}

	for _, args := range helpers {
		if _, err := exec.LookPath(args[0]); err == nil {
			out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s failed: %s\n%s", args[0], err, string(out))
			}
			return nil
		}
	}

	return fmt.Errorf("no AUR helper found (yay or paru required)")
}
