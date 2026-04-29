package packages

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/hthienloc/fcitx5-lotus-installer/internal/distro"
)

type Deps struct {
	Base        []string
	Fcitx5Dev   []string
	BuildTools  []string
	GoDeps      []string
}

func GetBuildDeps(dt distro.DistroType) Deps {
	switch dt {
	case distro.Arch:
		return Deps{
			Base:       []string{"base-devel", "cmake", "extra-cmake-modules", "golang", "pkgconf", "hicolor-icon-theme"},
			Fcitx5Dev:  []string{"fcitx5", "fcitx5-configtool", "fmtlib", "libinput", "libx11"},
			BuildTools: []string{"git", "cmake", "make", "gcc"},
			GoDeps:     []string{"go"},
		}
	case distro.Debian, distro.Ubuntu:
		return Deps{
			Base:       []string{"build-essential", "cmake", "extra-cmake-modules", "golang-go", "pkg-config", "hicolor-icon-theme"},
			Fcitx5Dev:  []string{"libfcitx5core-dev", "libfcitx5config-dev", "libfcitx5utils-dev", "fcitx5-modules-dev", "libinput-dev", "libudev-dev", "libx11-dev"},
			BuildTools: []string{"git", "cmake", "make", "gcc", "g++"},
			GoDeps:     []string{"golang-go"},
		}
	case distro.Fedora:
		return Deps{
			Base:       []string{"cmake", "extra-cmake-modules", "golang", "pkg-config", "hicolor-icon-theme"},
			Fcitx5Dev:  []string{"fcitx5-devel", "libinput-devel", "libudev-devel", "libX11-devel"},
			BuildTools: []string{"git", "cmake", "make", "gcc", "gcc-c++"},
			GoDeps:     []string{"golang"},
		}
	case distro.OpenSUSE:
		return Deps{
			Base:       []string{"cmake", "extra-cmake-modules", "go", "pkg-config", "hicolor-icon-theme"},
			Fcitx5Dev:  []string{"fcitx5-devel", "libinput-devel", "libudev-devel", "libX11-devel"},
			BuildTools: []string{"git", "cmake", "make", "gcc", "gcc-c++"},
			GoDeps:     []string{"go"},
		}
	default:
		return Deps{}
	}
}

func AllDeps(dt distro.DistroType) []string {
	deps := GetBuildDeps(dt)
	all := append(deps.Base, deps.Fcitx5Dev...)
	all = append(all, deps.BuildTools...)
	all = append(all, deps.GoDeps...)
	return dedupe(all)
}

func dedupe(s []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func CheckInstalled(pkg string) bool {
	cmd := exec.Command("which", pkg)
	return cmd.Run() == nil
}

func IsPackageInstalled(pkg string, dt distro.DistroType) bool {
	var cmd *exec.Cmd
	switch dt {
	case distro.Arch:
		cmd = exec.Command("pacman", "-Q", pkg)
	case distro.Debian, distro.Ubuntu:
		cmd = exec.Command("dpkg-query", "-W", "-f=${Status}", pkg)
	case distro.Fedora:
		cmd = exec.Command("rpm", "-q", pkg)
	case distro.OpenSUSE:
		cmd = exec.Command("rpm", "-q", pkg)
	default:
		return false
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	if dt == distro.Debian || dt == distro.Ubuntu {
		return strings.Contains(string(out), "install ok installed")
	}
	return true
}

func InstallPackages(packages []string, d distro.DistroInfo) error {
	if len(packages) == 0 {
		return nil
	}

	args := strings.Fields(d.InstallCmd)
	args = append(args, packages...)

	cmd := exec.Command("sudo", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	fmt.Printf("Installing %d packages...\n", len(packages))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install packages: %s\n%s", err, string(out))
	}
	return nil
}

func InstallSingle(pkg string, d distro.DistroInfo) error {
	return InstallPackages([]string{pkg}, d)
}
