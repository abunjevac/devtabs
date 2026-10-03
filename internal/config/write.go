package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Marshal renders cfg as YAML, omitting empty fields and separating tabs with blank lines.
func Marshal(cfg *Config) ([]byte, error) {
	general := *cfg

	general.Tabs = nil

	head, err := encode(&general)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	if strings.TrimSpace(head) != "{}" {
		buf.WriteString(head)
		buf.WriteString("\n")
	}

	buf.WriteString("tabs:\n")

	for i, tab := range cfg.Tabs {
		item, err := encode([]TabConfig{tab})
		if err != nil {
			return nil, err
		}

		if i > 0 {
			buf.WriteString("\n")
		}

		buf.WriteString(indent(item, "  "))
	}

	return buf.Bytes(), nil
}

func encode(v any) (string, error) {
	var buf bytes.Buffer

	enc := yaml.NewEncoder(&buf)

	enc.SetIndent(2)

	if err := enc.Encode(v); err != nil {
		return "", fmt.Errorf("encode-config: %w", err)
	}

	if err := enc.Close(); err != nil {
		return "", fmt.Errorf("encode-config: %w", err)
	}

	return buf.String(), nil
}

// indent prefixes every non-empty line, which keeps block scalars valid.
func indent(s, prefix string) string {
	lines := strings.SplitAfter(s, "\n")

	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = prefix + line
		}
	}

	return strings.Join(lines, "")
}

// HasComments reports whether the YAML document contains any comments.
// Invalid YAML reports false.
func HasComments(data string) bool {
	var root yaml.Node

	if err := yaml.Unmarshal([]byte(data), &root); err != nil {
		return false
	}

	return nodeHasComments(&root)
}

func nodeHasComments(n *yaml.Node) bool {
	if n.HeadComment != "" || n.LineComment != "" || n.FootComment != "" {
		return true
	}

	return slices.ContainsFunc(n.Content, nodeHasComments)
}

// Save writes cfg to path atomically, keeping the existing file mode when the file exists.
func Save(path string, cfg *Config) error {
	data, err := Marshal(cfg)
	if err != nil {
		return err
	}

	mode := fs.FileMode(0o644)

	info, err := os.Stat(path)

	switch {
	case err == nil:
		mode = info.Mode().Perm()
	case !errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("stat-config: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("create-temp: %w", err)
	}

	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()

		return fmt.Errorf("write-config: %w", err)
	}

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()

		return fmt.Errorf("chmod-config: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close-config: %w", err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("rename-config: %w", err)
	}

	return nil
}
