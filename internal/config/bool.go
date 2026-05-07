package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Bool accepts true/false/yes/no/on/off/1/0 (case-insensitive).
// yaml.v3 follows YAML 1.2 where "yes"/"on" are plain strings, not booleans.
type Bool bool

func (b *Bool) UnmarshalYAML(value *yaml.Node) error {
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
