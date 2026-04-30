package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	RepoURL     = "https://github.com/LotusInputMethod/fcitx5-lotus.git"
	InstallDir  = "/usr"
)

type Builder struct {
	WorkDir    string
	SourceDir  string
	BuildDir   string
	LibDir     string
	Jobs       int
}

func NewBuilder(workDir string) *Builder {
	libDir := "/usr/lib"
	if runtime.GOARCH == "arm64" || runtime.GOARCH == "aarch64" {
		libDir = "/usr/lib"
	}

	return &Builder{
		WorkDir:   workDir,
		SourceDir: filepath.Join(workDir, "fcitx5-lotus"),
		BuildDir:  filepath.Join(workDir, "fcitx5-lotus", "build"),
		LibDir:    libDir,
		Jobs:      runtime.NumCPU(),
	}
}

func (b *Builder) Clone() error {
	if _, err := os.Stat(b.SourceDir); err == nil {
		fmt.Println("Source directory already exists, ensuring submodules are up to date...")
		return b.UpdateSubmodules()
	}

	fmt.Println("Cloning fcitx5-lotus repository...")
	cmd := exec.Command("git", "clone", "--depth", "1", "--recursive", RepoURL, b.SourceDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *Builder) UpdateSubmodules() error {
	cmd := exec.Command("git", "-C", b.SourceDir, "submodule", "update", "--init", "--recursive")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *Builder) PullLatest() error {
	if _, err := os.Stat(b.SourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist")
	}

	fmt.Println("Pulling latest changes...")
	cmd := exec.Command("git", "-C", b.SourceDir, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *Builder) Configure() error {
	if err := os.MkdirAll(b.BuildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build dir: %w", err)
	}

	fmt.Println("Configuring cmake...")
	cmd := exec.Command("cmake",
		"-DCMAKE_INSTALL_PREFIX="+InstallDir,
		"-DCMAKE_INSTALL_LIBDIR="+b.LibDir,
		"-S", b.SourceDir,
		"-B", b.BuildDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *Builder) Build() error {
	fmt.Printf("Building with %d jobs...\n", b.Jobs)
	cmd := exec.Command("cmake",
		"--build", b.BuildDir,
		"-j", fmt.Sprintf("%d", b.Jobs),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *Builder) Install() error {
	fmt.Println("Installing fcitx5-lotus...")
	cmd := exec.Command("sudo", "cmake", "--install", b.BuildDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *Builder) FullBuild() error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Clone", b.Clone},
		{"Configure", b.Configure},
		{"Build", b.Build},
		{"Install", b.Install},
	}

	for _, step := range steps {
		fmt.Printf("\n[build] Step: %s\n", step.name)
		if err := step.fn(); err != nil {
			return fmt.Errorf("step %s failed: %w", step.name, err)
		}
	}

	return nil
}

func (b *Builder) Cleanup() error {
	fmt.Println("Cleaning up build artifacts...")
	return os.RemoveAll(b.BuildDir)
}
