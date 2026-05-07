package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration for YAML parsing of Go duration strings.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	dur, err := time.ParseDuration(value.Value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", value.Value, err)
	}

	if dur < 0 {
		return fmt.Errorf("startup_delay must be >= 0, got %s", value.Value)
	}

	d.Duration = dur

	return nil
}

// FlexBool accepts true/false/yes/no/on/off/1/0 (case-insensitive).
// yaml.v3 follows YAML 1.2 where "yes"/"on" are plain strings, not booleans.
type FlexBool bool

func (b *FlexBool) UnmarshalYAML(value *yaml.Node) error {
	switch strings.ToLower(value.Value) {
	case "true", "yes", "on", "1":
		*b = true
	case "false", "no", "off", "0":
		*b = false
	default:
		return fmt.Errorf("invalid boolean value %q: use true/false/yes/no/on/off/1/0", value.Value)
	}

	return nil
}

// TabConfig holds the configuration for a single tab.
type TabConfig struct {
	Name         string   `yaml:"name"`
	Command      string   `yaml:"command"`
	WorkingDir   string   `yaml:"working_dir"`
	RunOnStartup FlexBool `yaml:"run_on_startup"`
	StartupDelay Duration `yaml:"startup_delay"`
	Shell        string   `yaml:"shell"`
	ShellArgs    []string `yaml:"shell_args"`
}

// Config is the top-level configuration.
type Config struct {
	StartupTab   string      `yaml:"startup_tab"`
	Font         string      `yaml:"font"`
	FontSize     float64     `yaml:"font_size"`
	WindowWidth  int         `yaml:"window_width"`
	WindowHeight int         `yaml:"window_height"`
	Tabs         []TabConfig `yaml:"tabs"`
}

// Load reads and validates a config file from the given path.
// root is used to resolve relative working_dir values.
func Load(path, root string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	cfg, err := LoadBytes(data)
	if err != nil {
		return nil, err
	}

	applyDefaults(cfg, root)

	if err := validatePaths(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadBytes parses and validates config from raw YAML bytes.
// working_dir paths are not resolved; call Load for full validation.
func LoadBytes(data []byte) (*Config, error) {
	var cfg Config

	dec := yaml.NewDecoder(strings.NewReader(string(data)))

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
	if cfg.Font == "" {
		cfg.Font = "Monospace"
	}

	if cfg.FontSize == 0 {
		cfg.FontSize = 12
	}

	if cfg.WindowWidth == 0 {
		cfg.WindowWidth = 1200
	}

	if cfg.WindowHeight == 0 {
		cfg.WindowHeight = 800
	}

	for i := range cfg.Tabs {
		tab := &cfg.Tabs[i]

		if tab.Shell == "" {
			tab.Shell = "/bin/zsh"
		}

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
