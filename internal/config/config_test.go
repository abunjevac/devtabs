package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/abunjevac/devtabs/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		input string
		want  string
	}{
		{"~/projects/app", home + "/projects/app"},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			cfg, err := config.LoadFromString("tabs:\n  - name: t\n    command: x\n    working_dir: " + tc.input)
			require.NoError(t, err)

			assert.True(t, strings.HasPrefix(cfg.Tabs[0].WorkingDir, tc.want),
				"got %q, want prefix %q", cfg.Tabs[0].WorkingDir, tc.want)
		})
	}
}

func TestValidate(t *testing.T) {
	t.Run("missing name", func(t *testing.T) {
		_, err := config.LoadFromString("tabs:\n  - command: foo\n")
		assert.Error(t, err)
	})

	t.Run("missing command", func(t *testing.T) {
		_, err := config.LoadFromString("tabs:\n  - name: foo\n")
		assert.Error(t, err)
	})

	t.Run("duplicate names", func(t *testing.T) {
		_, err := config.LoadFromString("tabs:\n  - name: a\n    command: x\n  - name: a\n    command: y\n")
		assert.Error(t, err)
	})

	t.Run("startup_tab not found", func(t *testing.T) {
		_, err := config.LoadFromString("startup_tab: missing\ntabs:\n  - name: a\n    command: x\n")
		assert.Error(t, err)
	})

	t.Run("empty tabs", func(t *testing.T) {
		_, err := config.LoadFromString("tabs: []\n")
		assert.Error(t, err)
	})

	t.Run("unknown field rejected", func(t *testing.T) {
		_, err := config.LoadFromString("tabs:\n  - name: a\n    command: x\n    unknown_field: oops\n")
		assert.Error(t, err)
	})

	t.Run("valid minimal", func(t *testing.T) {
		cfg, err := config.LoadFromString("tabs:\n  - name: api\n    command: go run .\n")
		require.NoError(t, err)

		require.Len(t, cfg.Tabs, 1)
		assert.Equal(t, "/bin/zsh", cfg.Tabs[0].Shell)
		assert.Equal(t, []string{"-l"}, cfg.Tabs[0].ShellArgs)
	})

	t.Run("valid with startup_tab", func(t *testing.T) {
		cfg, err := config.LoadFromString("startup_tab: api\ntabs:\n  - name: api\n    command: go run .\n")
		require.NoError(t, err)

		assert.Equal(t, "api", cfg.StartupTab)
	})
}
