package services

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

type InitSystem string

const (
	Systemd InitSystem = "systemd"
	OpenRC  InitSystem = "openrc"
	Runit   InitSystem = "runit"
)

type ServiceManager struct {
	Init InitSystem
	De   string
	Env  string
}

func New(init InitSystem, de, env string) *ServiceManager {
	return &ServiceManager{
		Init: init,
		De:   de,
		Env:  env,
	}
}

func (sm *ServiceManager) CreateUserAndGroup() error {
	fmt.Println("Creating uinput_proxy user and input group...")

	cmd := exec.Command("sudo", "groupadd", "-f", "input")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create group: %s", string(out))
	}

	cmd = exec.Command("sudo", "useradd", "-M", "-g", "input", "-s", "/usr/bin/nologin", "-d", "/", "uinput_proxy")
	if out, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "already exists") {
			return fmt.Errorf("failed to create user: %s", string(out))
		}
	}

	fmt.Println("uinput_proxy user created")
	return nil
}

func (sm *ServiceManager) ReloadUdev() error {
	fmt.Println("Reloading udev rules...")

	cmd := exec.Command("sudo", "udevadm", "control", "--reload-rules")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("udev reload failed: %s", string(out))
	}

	cmd = exec.Command("sudo", "udevadm", "trigger")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("udev trigger failed: %s", string(out))
	}

	fmt.Println("Udev rules reloaded")
	return nil
}

func (sm *ServiceManager) ModprobeUinput() error {
	fmt.Println("Loading uinput kernel module...")

	cmd := exec.Command("sudo", "modprobe", "uinput")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("modprobe uinput failed: %s", string(out))
	}

	fmt.Println("uinput module loaded")
	return nil
}

func (sm *ServiceManager) ActivateServer() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("cannot get current user: %w", err)
	}
	username := currentUser.Username

	fmt.Println("Activating fcitx5-lotus-server service...")

	switch sm.Init {
	case Systemd:
		cmd := exec.Command("sudo", "systemctl", "enable", "--now", "fcitx5-lotus-server@"+username+".service")
		out, err := cmd.CombinedOutput()
		if err != nil {
			cmd2 := exec.Command("sudo", "systemd-sysusers")
			cmd2.Run()
			cmd3 := exec.Command("sudo", "systemctl", "enable", "--now", "fcitx5-lotus-server@"+username+".service")
			out, err = cmd3.CombinedOutput()
			if err != nil {
				return fmt.Errorf("systemctl enable failed: %s", string(out))
			}
		}
	case OpenRC:
		svcName := "fcitx5-lotus." + username

		cmd := exec.Command("sudo", "ln", "-sf", "/etc/init.d/fcitx5-lotus", "/etc/init.d/"+svcName)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("openrc symlink failed: %s", string(out))
		}

		cmd = exec.Command("sudo", "rc-update", "add", svcName, "default")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("rc-update add failed: %s", string(out))
		}

		cmd = exec.Command("sudo", "rc-service", svcName, "restart")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("rc-service restart failed: %s", string(out))
		}
	case Runit:
		svcName := "fcitx5-lotus." + username

		var svDir string
		if _, err := os.Stat("/etc/sv"); err == nil {
			svDir = "/etc/sv"
		} else if _, err := os.Stat("/etc/runit/sv"); err == nil {
			svDir = "/etc/runit/sv"
		} else {
			return fmt.Errorf("runit service directory not found")
		}

		var serviceDir string
		if _, err := os.Stat("/var/service"); err == nil {
			serviceDir = "/var/service"
		} else if _, err := os.Stat("/run/runit/service"); err == nil {
			serviceDir = "/run/runit/service"
		} else if _, err := os.Stat("/etc/runit/runsvdir/current"); err == nil {
			serviceDir = "/etc/runit/runsvdir/current"
		} else {
			return fmt.Errorf("runit runsvdir not found")
		}

		cmd := exec.Command("sudo", "ln", "-sf", svDir+"/fcitx5-lotus", serviceDir+"/"+svcName)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("runit symlink failed: %s", string(out))
		}

		cmd = exec.Command("sudo", "sv", "start", serviceDir+"/"+svcName)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("sv start failed: %s", string(out))
		}
	}

	fmt.Println("fcitx5-lotus-server activated")
	return nil
}

func (sm *ServiceManager) KillIBus() error {
	fmt.Println("Checking for IBus processes...")

	cmd := exec.Command("pkill", "-x", "ibus-daemon")
	cmd.Run()

	cmd = exec.Command("pkill", "-x", "ibus")
	cmd.Run()

	fmt.Println("IBus processes killed (if any)")
	return nil
}

func (sm *ServiceManager) DBusFixForNonSystemd() (string, error) {
	if sm.Env == "Wayland" && (sm.Init == OpenRC || sm.Init == Runit) {
		return "DBUS_SESSION_BUS_ADDRESS=unix:path=$XDG_RUNTIME_DIR/bus", nil
	}
	return "", nil
}

func (sm *ServiceManager) ChromiumWaylandFlags() string {
	if sm.Env == "Wayland" {
		if sm.De == "KDE Plasma" {
			return "--enable-features=UseOzonePlatform --ozone-platform=wayland --enable-wayland-ime"
		}
		return "--enable-features=UseOzonePlatform --ozone-platform=wayland --enable-wayland-ime --wayland-text-input-version=3"
	}
	return ""
}

func (sm *ServiceManager) SetupAllForSource() error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Create user/group", sm.CreateUserAndGroup},
		{"Reload udev", sm.ReloadUdev},
		{"Modprobe uinput", sm.ModprobeUinput},
		{"Activate server", sm.ActivateServer},
		{"Kill IBus", sm.KillIBus},
	}

	for _, step := range steps {
		fmt.Printf("[services] %s\n", step.name)
		if err := step.fn(); err != nil {
			return fmt.Errorf("step %s failed: %w", step.name, err)
		}
	}

	return nil
}
