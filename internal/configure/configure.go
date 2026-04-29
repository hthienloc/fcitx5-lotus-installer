package configure

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	EnvConfDir  = ".config/environment.d"
	EnvConfFile = "90-fcitx5-lotus.conf"
	Fcitx5ConfDir = ".config/fcitx5"
)

type Configurer struct {
	HomeDir string
}

func NewConfigurer() (*Configurer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot get home dir: %w", err)
	}
	return &Configurer{HomeDir: home}, nil
}

func (c *Configurer) SetupEnvironment() error {
	confDir := filepath.Join(c.HomeDir, EnvConfDir)
	confPath := filepath.Join(confDir, EnvConfFile)

	if err := os.MkdirAll(confDir, 0755); err != nil {
		return fmt.Errorf("failed to create env conf dir: %w", err)
	}

	envVars := []string{
		"GTK_IM_MODULE=fcitx",
		"QT_IM_MODULE=fcitx",
		"XMODIFIERS=@im=fcitx",
		"SDL_IM_MODULE=fcitx",
		"GLFW_IM_MODULE=ibus",
	}

	content := strings.Join(envVars, "\n") + "\n"

	if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write env conf: %w", err)
	}

	fmt.Printf("Created %s\n", confPath)
	return nil
}

func (c *Configurer) SetupFcitx5Config() error {
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
		fmt.Printf("Created fcitx5 profile at %s\n", profilePath)
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

	fmt.Println("fcitx5 restarted successfully")
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
		{"Environment", c.SetupEnvironment},
		{"Fcitx5 Config", c.SetupFcitx5Config},
	}

	for _, step := range steps {
		fmt.Printf("[configure] %s\n", step.name)
		if err := step.fn(); err != nil {
			fmt.Printf("[configure] Warning: %s failed: %v\n", step.name, err)
		}
	}

	return nil
}
