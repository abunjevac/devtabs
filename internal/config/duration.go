package config

import (
	"fmt"
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
