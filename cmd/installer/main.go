package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/build"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/configure"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/distro"
	"github.com/hthienloc/fcitx5-lotus-installer/internal/packages"
)

type step int

const (
	stepWelcome step = iota
	stepDetect
	stepCheckDeps
	stepInstallDeps
	stepClone
	stepBuild
	stepInstall
	stepConfigure
	stepDone
	stepError
)

var stepList = []struct {
	name string
	s    step
}{
	{"Welcome", stepWelcome},
	{"Detect OS", stepDetect},
	{"Check Dependencies", stepCheckDeps},
	{"Install Dependencies", stepInstallDeps},
	{"Clone Source", stepClone},
	{"Build", stepBuild},
	{"Install", stepInstall},
	{"Configure", stepConfigure},
	{"Complete", stepDone},
}

type stepMsg struct {
	step   step
	status string
	err    error
}

type model struct {
	currentStep step
	status      string
	err         error
	distro      distro.DistroInfo
	missingDeps []string
	workDir     string
	quitting    bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter", " ":
			if m.currentStep == stepWelcome {
				return m, m.advance()
			}
		}
	case stepMsg:
		m.currentStep = msg.step
		m.status = msg.status
		m.err = msg.err
		if msg.err != nil {
			m.currentStep = stepError
			return m, tea.Quit
		}
		if msg.step == stepDone {
			m.quitting = true
			return m, tea.Quit
		}
		return m, m.advance()
	}
	return m, nil
}

func (m *model) advance() tea.Cmd {
	return func() tea.Msg {
		switch m.currentStep {
		case stepWelcome:
			return stepMsg{step: stepDetect, status: "Detecting system..."}
		case stepDetect:
			return m.runDetect()
		case stepCheckDeps:
			return m.runCheckDeps()
		case stepInstallDeps:
			return m.runInstallDeps()
		case stepClone:
			return m.runClone()
		case stepBuild:
			return m.runBuild()
		case stepInstall:
			return m.runInstall()
		case stepConfigure:
			return m.runConfigure()
		case stepDone:
			return stepMsg{step: stepDone, status: "Installation complete!"}
		}
		return nil
	}
}

func (m *model) runDetect() stepMsg {
	d, err := distro.Detect()
	if err != nil {
		return stepMsg{step: stepError, err: fmt.Errorf("OS detection failed: %w", err)}
	}
	m.distro = d
	return stepMsg{step: stepCheckDeps, status: fmt.Sprintf("Detected: %s", d)}
}

func (m *model) runCheckDeps() stepMsg {
	allDeps := packages.AllDeps(m.distro.Type)
	var missing []string
	for _, pkg := range allDeps {
		if !packages.IsPackageInstalled(pkg, m.distro.Type) {
			missing = append(missing, pkg)
		}
	}
	m.missingDeps = missing
	if len(missing) == 0 {
		return stepMsg{step: stepClone, status: "All dependencies satisfied!"}
	}
	return stepMsg{step: stepInstallDeps, status: fmt.Sprintf("Found %d missing packages", len(missing))}
}

func (m *model) runInstallDeps() stepMsg {
	if len(m.missingDeps) == 0 {
		return stepMsg{step: stepClone, status: "No dependencies to install"}
	}
	err := packages.InstallPackages(m.missingDeps, m.distro)
	if err != nil {
		return stepMsg{step: stepError, err: fmt.Errorf("dependency installation failed: %w", err)}
	}
	return stepMsg{step: stepClone, status: "Dependencies installed successfully"}
}

func (m *model) runClone() stepMsg {
	home, err := os.UserHomeDir()
	if err != nil {
		return stepMsg{step: stepError, err: fmt.Errorf("cannot get home dir: %w", err)}
	}
	m.workDir = filepath.Join(home, ".cache", "fcitx5-lotus-installer")
	os.MkdirAll(m.workDir, 0755)

	b := build.NewBuilder(m.workDir)
	if err := b.Clone(); err != nil {
		return stepMsg{step: stepError, err: fmt.Errorf("clone failed: %w", err)}
	}
	return stepMsg{step: stepBuild, status: "Source cloned successfully"}
}

func (m *model) runBuild() stepMsg {
	b := build.NewBuilder(m.workDir)
	if err := b.Configure(); err != nil {
		return stepMsg{step: stepError, err: fmt.Errorf("cmake configure failed: %w", err)}
	}
	if err := b.Build(); err != nil {
		return stepMsg{step: stepError, err: fmt.Errorf("build failed: %w", err)}
	}
	return stepMsg{step: stepInstall, status: "Build completed successfully"}
}

func (m *model) runInstall() stepMsg {
	b := build.NewBuilder(m.workDir)
	if err := b.Install(); err != nil {
		return stepMsg{step: stepError, err: fmt.Errorf("installation failed: %w", err)}
	}
	return stepMsg{step: stepConfigure, status: "Installed to system"}
}

func (m *model) runConfigure() stepMsg {
	c, err := configure.NewConfigurer()
	if err != nil {
		return stepMsg{step: stepError, err: fmt.Errorf("configurer init failed: %w", err)}
	}
	if err := c.ApplyAll(); err != nil {
		return stepMsg{step: stepError, err: fmt.Errorf("configuration failed: %w", err)}
	}
	return stepMsg{step: stepDone, status: "Configuration applied"}
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("   ======================================\n")
	b.WriteString("   🪷  fcitx5-lotus Installer  🪷\n")
	b.WriteString("   ======================================\n\n")

	for _, s := range stepList {
		prefix := "  "
		switch {
		case m.currentStep > s.s:
			prefix = "✅ "
		case m.currentStep == s.s:
			prefix = "🔹 "
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", prefix, s.name))
	}

	b.WriteString("\n")

	if m.status != "" {
		b.WriteString(fmt.Sprintf("  %s\n", m.status))
	}
	if m.err != nil {
		b.WriteString(fmt.Sprintf("  ❌ Error: %v\n", m.err))
	}

	if m.currentStep == stepWelcome {
		b.WriteString("\n  Press Enter to start installation\n")
	}

	b.WriteString("\n  Press q to quit\n")
	return b.String()
}

func runTUI() {
	m := model{currentStep: stepWelcome}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func runCLI() {
	fmt.Printf("💻 OS: %s\n", runtime.GOOS)
	fmt.Printf("🏗️  Arch: %s\n", runtime.GOARCH)
	fmt.Printf("🔍 Detecting System Environment...\n")

	d, err := distro.Detect()
	if err != nil {
		fmt.Printf("❌ Failed to detect OS: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Detected: %s\n\n", d)

	fmt.Println("📦 Checking dependencies...")
	allDeps := packages.AllDeps(d.Type)
	var missing []string
	for _, pkg := range allDeps {
		if !packages.IsPackageInstalled(pkg, d.Type) {
			missing = append(missing, pkg)
		}
	}

	if len(missing) > 0 {
		fmt.Printf("⚠️  Missing %d packages: %v\n", len(missing), missing)
		fmt.Printf("📥 Installing dependencies...\n")
		if err := packages.InstallPackages(missing, d); err != nil {
			fmt.Printf("❌ Failed to install dependencies: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Dependencies installed\n")
	} else {
		fmt.Println("✅ All dependencies satisfied\n")
	}

	home, _ := os.UserHomeDir()
	workDir := filepath.Join(home, ".cache", "fcitx5-lotus-installer")
	os.MkdirAll(workDir, 0755)

	b := build.NewBuilder(workDir)

	fmt.Println("📥 Cloning fcitx5-lotus...")
	if err := b.Clone(); err != nil {
		fmt.Printf("❌ Clone failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🔨 Configuring...")
	if err := b.Configure(); err != nil {
		fmt.Printf("❌ Configure failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🔨 Building...")
	if err := b.Build(); err != nil {
		fmt.Printf("❌ Build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("📤 Installing...")
	if err := b.Install(); err != nil {
		fmt.Printf("❌ Install failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("⚙️  Configuring environment...")
	cfg, err := configure.NewConfigurer()
	if err != nil {
		fmt.Printf("❌ Config failed: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.ApplyAll(); err != nil {
		fmt.Printf("⚠️  Configuration warning: %v\n", err)
	}

	if cfg.CheckFcitx5Running() {
		fmt.Println("🔄 Restarting fcitx5...")
		if err := cfg.RestartFcitx5(); err != nil {
			fmt.Printf("⚠️  Could not restart fcitx5: %v\n", err)
			fmt.Println("   Please restart fcitx5 manually: fcitx5 -d --replace")
		}
	}

	fmt.Println("\n✅ Installation complete!")
	fmt.Println("\n📝 Next steps:")
	fmt.Println("  1. Log out and log back in (or restart)")
	fmt.Println("  2. Add 'Lotus' input method in fcitx5 config")
	fmt.Println("  3. Start typing!")
	fmt.Println("\n🔧 Manual restart: fcitx5 -d --replace")
	fmt.Println("📖 Docs: https://lotusinputmethod.github.io/")

	b.Cleanup()
}

func main() {
	if os.Geteuid() == 0 {
		fmt.Println("❌ Error: Please do not run as root.")
		fmt.Println("   The installer will ask for sudo when needed.")
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "--cli" {
		runCLI()
	} else {
		runTUI()
	}
}
