package distro

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type DistroType string

const (
	Unknown    DistroType = "unknown"
	Arch       DistroType = "arch"
	Debian     DistroType = "debian"
	Ubuntu     DistroType = "ubuntu"
	Fedora     DistroType = "fedora"
	OpenSUSE   DistroType = "opensuse"
)

type DistroInfo struct {
	Type       DistroType
	Name       string
	Version    string
	IDLike     string
	PkgManager string
	InstallCmd string
	SudoCmd    string
}

func Detect() (DistroInfo, error) {
	info := DistroInfo{
		Type:   Unknown,
		Name:   "Unknown",
	}

	f, err := os.Open("/etc/os-release")
	if err != nil {
		return info, fmt.Errorf("cannot detect OS: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id := strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			info.Type = classifyDistro(id)
		}
		if strings.HasPrefix(line, "NAME=") {
			info.Name = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			info.Version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
		if strings.HasPrefix(line, "ID_LIKE=") {
			info.IDLike = strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), "\"")
		}
	}

	if info.Type == Unknown {
		if strings.Contains(info.IDLike, "arch") {
			info.Type = Arch
		} else if strings.Contains(info.IDLike, "debian") {
			info.Type = Debian
		} else if strings.Contains(info.IDLike, "fedora") {
			info.Type = Fedora
		} else if strings.Contains(info.IDLike, "suse") {
			info.Type = OpenSUSE
		}
	}

	info.PkgManager, info.InstallCmd, info.SudoCmd = getPackageManager(info.Type)

	return info, nil
}

func classifyDistro(id string) DistroType {
	switch {
	case id == "arch", id == "archarm", id == "endeavouros", id == "manjaro", id == "cachyos":
		return Arch
	case id == "debian", id == "linuxmint":
		return Debian
	case id == "ubuntu", id == "pop", id == "neon":
		return Ubuntu
	case id == "fedora", id == "nobara":
		return Fedora
	case strings.HasPrefix(id, "opensuse"):
		return OpenSUSE
	default:
		return Unknown
	}
}

func getPackageManager(dt DistroType) (manager, installCmd, sudoCmd string) {
	switch dt {
	case Arch:
		return "pacman", "pacman -Sy --noconfirm", "sudo"
	case Debian, Ubuntu:
		return "apt", "apt-get install -y", "sudo"
	case Fedora:
		return "dnf", "dnf install -y", "sudo"
	case OpenSUSE:
		return "zypper", "zypper install -y", "sudo"
	default:
		return "", "", ""
	}
}

func (d DistroType) String() string {
	return string(d)
}

func (d DistroInfo) String() string {
	return fmt.Sprintf("%s %s (pkg: %s)", d.Name, d.Version, d.PkgManager)
}
