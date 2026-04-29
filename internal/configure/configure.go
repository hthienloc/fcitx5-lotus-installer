package configure

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	EnvConfDir    = ".config/environment.d"
	EnvConfFile   = "90-fcitx5-lotus.conf"
	Fcitx5ConfDir = ".config/fcitx5"
)

type ShellType string

const (
	Bash ShellType = "bash"
	Zsh  ShellType = "zsh"
	Fish ShellType = "fish"
)

type DesktopEnv string

const (
	GNOME      DesktopEnv = "GNOME"
	KDEPlasma  DesktopEnv = "KDE Plasma"
	Xfce       DesktopEnv = "Xfce"
	Cinnamon   DesktopEnv = "Cinnamon"
	MATE       DesktopEnv = "MATE"
	Pantheon   DesktopEnv = "Pantheon"
	Budgie     DesktopEnv = "Budgie"
	LXQt       DesktopEnv = "LXQt"
	COSMIC     DesktopEnv = "COSMIC"
	I3         DesktopEnv = "i3"
	Sway       DesktopEnv = "Sway"
	Hyprland   DesktopEnv = "Hyprland"
)

type SessionEnv string

const (
	X11    SessionEnv = "X11"
	Wayland SessionEnv = "Wayland"
)

type Configurer struct {
	HomeDir string
	Shell   ShellType
	DE      DesktopEnv
	Session SessionEnv
}

func NewConfigurer(shell ShellType, de DesktopEnv, session SessionEnv) (*Configurer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot get home dir: %w", err)
	}
	return &Configurer{
		HomeDir: home,
		Shell:   shell,
		DE:      de,
		Session: session,
	}, nil
}

func (c *Configurer) getEnvVars() []string {
	if c.Session == Wayland {
		switch c.DE {
		case KDEPlasma:
			return []string{
				"XMODIFIERS=@im=fcitx",
				"GLFW_IM_MODULE=ibus",
			}
		case GNOME, Sway:
			return []string{
				"XMODIFIERS=@im=fcitx",
				"QT_IM_MODULE=fcitx",
				"QT_IM_MODULES=wayland;fcitx",
				"GLFW_IM_MODULE=ibus",
			}
		default:
			return c.defaultEnvVars()
		}
	}
	return c.defaultEnvVars()
}

func (c *Configurer) defaultEnvVars() []string {
	return []string{
		"GTK_IM_MODULE=fcitx",
		"QT_IM_MODULE=fcitx",
		"XMODIFIERS=@im=fcitx",
		"SDL_IM_MODULE=fcitx",
		"GLFW_IM_MODULE=ibus",
	}
}

func (c *Configurer) SetupEnvironmentD() error {
	confDir := filepath.Join(c.HomeDir, EnvConfDir)
	confPath := filepath.Join(confDir, EnvConfFile)

	if err := os.MkdirAll(confDir, 0755); err != nil {
		return fmt.Errorf("failed to create env conf dir: %w", err)
	}

	envVars := c.getEnvVars()
	content := strings.Join(envVars, "\n") + "\n"

	if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write env conf: %w", err)
	}

	fmt.Printf("Created %s\n", confPath)
	return nil
}

func (c *Configurer) SetupShellProfile() error {
	envVars := c.getEnvVars()
	var exportLines []string
	for _, v := range envVars {
		exportLines = append(exportLines, "export "+v)
	}

	switch c.Shell {
	case Bash:
		profilePath := filepath.Join(c.HomeDir, ".bash_profile")
		content := strings.Join(exportLines, "\n") + "\n"
		if err := appendToFile(profilePath, content); err != nil {
			return fmt.Errorf("bash_profile: %w", err)
		}
	case Zsh:
		profilePath := filepath.Join(c.HomeDir, ".zprofile")
		content := strings.Join(exportLines, "\n") + "\n"
		if err := appendToFile(profilePath, content); err != nil {
			return fmt.Errorf("zprofile: %w", err)
		}
	case Fish:
		fishConfigDir := filepath.Join(c.HomeDir, ".config/fish")
		fishConfigPath := filepath.Join(fishConfigDir, "config.fish")

		if err := os.MkdirAll(fishConfigDir, 0755); err != nil {
			return fmt.Errorf("fish config dir: %w", err)
		}

		var fishLines []string
		fishLines = append(fishLines, "if status is-login")
		for _, v := range envVars {
			parts := strings.SplitN(v, "=", 2)
			name := parts[0]
			val := ""
			if len(parts) > 1 {
				val = parts[1]
			}
			fishLines = append(fishLines, fmt.Sprintf("    set -Ux %s %s", name, val))
		}
		fishLines = append(fishLines, "end")

		content := strings.Join(fishLines, "\n") + "\n"
		if err := appendToFile(fishConfigPath, content); err != nil {
			return fmt.Errorf("fish config: %w", err)
		}
	}

	return nil
}

func (c *Configurer) SetupFcitx5Profile() error {
	confDir := filepath.Join(c.HomeDir, Fcitx5ConfDir)
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return fmt.Errorf("failed to create fcitx5 conf dir: %w", err)
	}

	profilePath := filepath.Join(confDir, "profile")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		profile := `[Groups/0]
Name=Default
Default Layout=us
DefaultIM=lotus

[Groups/0/Items/0]
Name=keyboard-us
Layout=

[Groups/0/Items/1]
Name=lotus
Layout=

[GroupOrder]
0=Default
`
		if err := os.WriteFile(profilePath, []byte(profile), 0644); err != nil {
			return fmt.Errorf("failed to write profile: %w", err)
		}
		fmt.Printf("Created fcitx5 profile\n")
	}

	return nil
}

func (c *Configurer) SetupAutostart() error {
	home := c.HomeDir
	configDir := filepath.Join(home, ".config")
	autostartDir := filepath.Join(configDir, "autostart")

	if err := os.MkdirAll(autostartDir, 0755); err != nil {
		return fmt.Errorf("autostart dir: %w", err)
	}

	desktopFile := `[Desktop Entry]
Type=Application
Name=Fcitx 5
Exec=fcitx5 -d
Comment=Fcitx5 Input Method
NoDisplay=true
X-GNOME-Autostart-Phase=InputMethods
X-GNOME-Provides=inputmethod
X-GNOME-Autostart-Notify=false
`
	desktopPath := filepath.Join(autostartDir, "fcitx5-autostart.desktop")
	if _, err := os.Stat(desktopPath); os.IsNotExist(err) {
		if err := os.WriteFile(desktopPath, []byte(desktopFile), 0644); err != nil {
			return fmt.Errorf("autostart desktop: %w", err)
		}
	}

	switch c.DE {
	case I3:
		i3config := filepath.Join(home, ".config/i3/config")
		if _, err := os.Stat(i3config); err == nil {
			f, err := os.OpenFile(i3config, os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString("\nexec --no-startup-id fcitx5 -d\n")
				f.Close()
			}
		}
	case Sway:
		swayconfig := filepath.Join(home, ".config/sway/config")
		if _, err := os.Stat(swayconfig); err == nil {
			f, err := os.OpenFile(swayconfig, os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString("\nexec --no-startup-id fcitx5 -d\n")
				f.Close()
			}
		}
	case Hyprland:
		hyprconfig := filepath.Join(home, ".config/hypr/hyprland.conf")
		if _, err := os.Stat(hyprconfig); err == nil {
			f, err := os.OpenFile(hyprconfig, os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString("\nexec-once = fcitx5 -d\n")
				f.Close()
			}
		}
	}

	return nil
}

func (c *Configurer) RestartFcitx5() error {
	fmt.Println("Restarting fcitx5...")

	cmd := exec.Command("pkill", "-x", "fcitx5")
	cmd.Run()

	cmd = exec.Command("fcitx5", "-d", "--replace")
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start fcitx5: %w", err)
	}

	fmt.Println("fcitx5 restarted")
	return nil
}

func (c *Configurer) CheckFcitx5Running() bool {
	cmd := exec.Command("pgrep", "-x", "fcitx5")
	return cmd.Run() == nil
}

func (c *Configurer) ApplyAll() error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Environment.d", c.SetupEnvironmentD},
		{"Shell profile", c.SetupShellProfile},
		{"Fcitx5 profile", c.SetupFcitx5Profile},
		{"Autostart", c.SetupAutostart},
	}

	for _, step := range steps {
		fmt.Printf("[configure] %s\n", step.name)
		if err := step.fn(); err != nil {
			fmt.Printf("[configure] Warning: %s failed: %v\n", step.name, err)
		}
	}

	return nil
}

func appendToFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
