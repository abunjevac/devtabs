package config

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/abunjevac/devtabs/internal/version"
)

// TabConfig holds the configuration for a single tab.
type TabConfig struct {
	Name         string   `yaml:"name"`
	Command      string   `yaml:"command"`
	WorkingDir   string   `yaml:"working_dir"`
	RunOnStartup Bool     `yaml:"run_on_startup"`
	StartupDelay Duration `yaml:"startup_delay"`
	Shell        string   `yaml:"shell"`
	ShellArgs    []string `yaml:"shell_args"`
	Profiles     []string `yaml:"profiles"`
}

// Config is the top-level configuration.
type Config struct {
	StartupTab        string      `yaml:"startup_tab"`
	WrapTabNavigation bool        `yaml:"wrap_tab_navigation"`
	Font              string      `yaml:"font"`
	FontSize          float64     `yaml:"font_size"`
	WindowWidth       int         `yaml:"window_width"`
	WindowHeight      int         `yaml:"window_height"`
	Title             string      `yaml:"title"`
	Terminal          string      `yaml:"terminal"`
	FileManager       string      `yaml:"file_manager"`
	Editor            string      `yaml:"editor"`
	Tabs              []TabConfig `yaml:"tabs"`
}

// Load reads and validates a config file from the given path.
// root is used to resolve relative working_dir values.
func Load(path, root string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	cfg, err := LoadFromString(string(data))
	if err != nil {
		return nil, err
	}

	applyDefaults(cfg, root)

	if err := validatePaths(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromString parses and validates config from a string.
// working_dir paths are not resolved; call Load for full validation.
func LoadFromString(data string) (*Config, error) {
	var cfg Config

	dec := yaml.NewDecoder(strings.NewReader(data))

	dec.KnownFields(true)

	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse-config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	applyDefaults(&cfg, "")

	return &cfg, nil
}

func applyDefaults(cfg *Config, root string) {
	cfg.Font = cmp.Or(cfg.Font, "Monospace")
	cfg.FontSize = cmp.Or(cfg.FontSize, 12)
	cfg.WindowWidth = cmp.Or(cfg.WindowWidth, 1200)
	cfg.WindowHeight = cmp.Or(cfg.WindowHeight, 800)
	cfg.Editor = cmp.Or(cfg.Editor, "zed")
	cfg.Title = cmp.Or(cfg.Title, "devtabs "+version.Version)

	for i := range cfg.Tabs {
		tab := &cfg.Tabs[i]

		tab.Shell = cmp.Or(tab.Shell, "/bin/zsh")

		if len(tab.ShellArgs) == 0 {
			tab.ShellArgs = []string{"-l"}
		}

		if tab.WorkingDir != "" {
			tab.WorkingDir = expandTilde(tab.WorkingDir)
		} else if root != "" {
			tab.WorkingDir = root
		}
	}
}

func validate(cfg *Config) error {
	if len(cfg.Tabs) == 0 {
		return errors.New("config must define at least one tab")
	}

	names := make(map[string]struct{}, len(cfg.Tabs))

	for i, tab := range cfg.Tabs {
		if tab.Name == "" {
			return fmt.Errorf("tab[%d]: name is required", i)
		}

		if tab.Command == "" {
			return fmt.Errorf("tab %q: command is required", tab.Name)
		}

		if _, dup := names[tab.Name]; dup {
			return fmt.Errorf("duplicate tab name: %q", tab.Name)
		}

		names[tab.Name] = struct{}{}
	}

	if cfg.StartupTab != "" {
		if _, ok := names[cfg.StartupTab]; !ok {
			return fmt.Errorf("startup_tab %q does not match any tab name", cfg.StartupTab)
		}
	}

	return nil
}

func expandTilde(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, path[1:])
}

// FilterByProfiles returns a copy of cfg with tabs filtered to those matching at
// least one of the requested profiles. Tabs with no profiles are always included.
// Returns an error if the filtered tab list would be empty.
// If profiles is nil or empty, cfg is returned unchanged.
func FilterByProfiles(cfg *Config, profiles []string) (*Config, error) {
	if len(profiles) == 0 {
		return cfg, nil
	}

	want := make(map[string]struct{}, len(profiles))

	for _, p := range profiles {
		want[p] = struct{}{}
	}

	filtered := make([]TabConfig, 0, len(cfg.Tabs))

	for _, tab := range cfg.Tabs {
		if tabMatchesProfiles(tab, want) {
			filtered = append(filtered, tab)
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no tabs match profiles: %s", strings.Join(profiles, ", "))
	}

	out := *cfg
	out.Tabs = filtered

	if out.StartupTab != "" && !tabNamesContain(filtered, out.StartupTab) {
		out.StartupTab = ""
	}

	return &out, nil
}

func tabMatchesProfiles(tab TabConfig, want map[string]struct{}) bool {
	if len(tab.Profiles) == 0 {
		return true
	}

	for _, p := range tab.Profiles {
		if _, ok := want[p]; ok {
			return true
		}
	}

	return false
}

func tabNamesContain(tabs []TabConfig, name string) bool {
	for _, tab := range tabs {
		if tab.Name == name {
			return true
		}
	}

	return false
}

func validatePaths(cfg *Config) error {
	for _, tab := range cfg.Tabs {
		if tab.WorkingDir == "" {
			continue
		}

		if _, err := os.Stat(tab.WorkingDir); err != nil {
			return fmt.Errorf("tab %q: working_dir %q: %w", tab.Name, tab.WorkingDir, err)
		}
	}

	return nil
}
