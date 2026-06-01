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

func TestProfiles(t *testing.T) {
	t.Run("profiles field parses", func(t *testing.T) {
		cfg, err := config.LoadFromString("tabs:\n  - name: a\n    command: x\n    profiles: [work, dev]\n")
		require.NoError(t, err)
		assert.Equal(t, []string{"work", "dev"}, cfg.Tabs[0].Profiles)
	})

	t.Run("no profiles field means always included", func(t *testing.T) {
		cfg, err := config.LoadFromString("tabs:\n  - name: a\n    command: x\n")
		require.NoError(t, err)
		assert.Empty(t, cfg.Tabs[0].Profiles)
	})
}

func TestApplicationDefaults(t *testing.T) {
	cfg, err := config.LoadFromString("tabs:\n  - name: a\n    command: x\n")
	require.NoError(t, err)

	assert.Empty(t, cfg.Terminal)
	assert.Empty(t, cfg.FileManager)
	assert.Equal(t, "zed", cfg.Editor)
}

func TestApplicationOverrides(t *testing.T) {
	cfg, err := config.LoadFromString("terminal: kitty\nfile_manager: nautilus\neditor: code\ntabs:\n  - name: a\n    command: x\n")
	require.NoError(t, err)

	assert.Equal(t, "kitty", cfg.Terminal)
	assert.Equal(t, "nautilus", cfg.FileManager)
	assert.Equal(t, "code", cfg.Editor)
}

func TestFilterByProfiles(t *testing.T) {
	base := func() *config.Config {
		cfg, err := config.LoadFromString(`tabs:
  - name: always
    command: x
  - name: work-only
    command: x
    profiles: [work]
  - name: dev-only
    command: x
    profiles: [dev]
  - name: multi
    command: x
    profiles: [work, dev]
`)
		require.NoError(t, err)
		return cfg
	}

	t.Run("empty profiles returns all tabs", func(t *testing.T) {
		cfg, err := config.FilterByProfiles(base(), nil)
		require.NoError(t, err)
		assert.Len(t, cfg.Tabs, 4)
	})

	t.Run("work profile includes always + work tabs", func(t *testing.T) {
		cfg, err := config.FilterByProfiles(base(), []string{"work"})
		require.NoError(t, err)
		assert.Equal(t, []string{"always", "work-only", "multi"}, tabNames(cfg))
	})

	t.Run("dev profile includes always + dev tabs", func(t *testing.T) {
		cfg, err := config.FilterByProfiles(base(), []string{"dev"})
		require.NoError(t, err)
		assert.Equal(t, []string{"always", "dev-only", "multi"}, tabNames(cfg))
	})

	t.Run("multiple profiles union", func(t *testing.T) {
		cfg, err := config.FilterByProfiles(base(), []string{"work", "dev"})
		require.NoError(t, err)
		assert.Len(t, cfg.Tabs, 4)
	})

	t.Run("no match returns error", func(t *testing.T) {
		cfg, err := config.LoadFromString(`tabs:
  - name: work-only
    command: x
    profiles: [work]
  - name: dev-only
    command: x
    profiles: [dev]
`)
		require.NoError(t, err)
		_, err = config.FilterByProfiles(cfg, []string{"nonexistent"})
		assert.Error(t, err)
	})

	t.Run("startup_tab cleared when filtered out", func(t *testing.T) {
		cfg, err := config.LoadFromString(`startup_tab: work-only
tabs:
  - name: always
    command: x
  - name: work-only
    command: x
    profiles: [work]
`)
		require.NoError(t, err)

		out, err := config.FilterByProfiles(cfg, []string{"dev"})
		require.NoError(t, err)
		assert.Empty(t, out.StartupTab)
	})

	t.Run("startup_tab preserved when still present", func(t *testing.T) {
		cfg, err := config.LoadFromString(`startup_tab: always
tabs:
  - name: always
    command: x
  - name: work-only
    command: x
    profiles: [work]
`)
		require.NoError(t, err)

		out, err := config.FilterByProfiles(cfg, []string{"work"})
		require.NoError(t, err)
		assert.Equal(t, "always", out.StartupTab)
	})
}

func tabNames(cfg *config.Config) []string {
	names := make([]string, len(cfg.Tabs))
	for i, tab := range cfg.Tabs {
		names[i] = tab.Name
	}
	return names
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
