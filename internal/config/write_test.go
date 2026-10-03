package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abunjevac/devtabs/internal/config"
)

func TestDecodeKeepsRawValues(t *testing.T) {
	cfg, err := config.Decode("tabs:\n  - name: a\n    command: x\n")
	require.NoError(t, err)

	assert.Empty(t, cfg.Font)
	assert.Empty(t, cfg.Tabs[0].Shell)
	assert.Empty(t, cfg.Tabs[0].ShellArgs)
}

func TestDecodeSkipsValidation(t *testing.T) {
	cfg, err := config.Decode("tabs:\n  - name: a\n")
	require.NoError(t, err)

	assert.Equal(t, "a", cfg.Tabs[0].Name)
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	_, err := config.Decode("bogus: 1\n")

	assert.Error(t, err)
}

func TestValidateFieldErrors(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		tab   int
		field string
	}{
		{"no tabs", "tabs: []\n", -1, "tabs"},
		{"global scrollback", "scrollback_lines: -2\ntabs:\n  - name: a\n    command: x\n", -1, "scrollback_lines"},
		{"missing name", "tabs:\n  - name: a\n    command: x\n  - command: y\n", 1, "name"},
		{"missing command", "tabs:\n  - name: a\n", 0, "command"},
		{"tab scrollback", "tabs:\n  - name: a\n    command: x\n    scrollback_lines: -5\n", 0, "scrollback_lines"},
		{"duplicate name", "tabs:\n  - name: a\n    command: x\n  - name: a\n    command: y\n", 1, "name"},
		{"startup tab", "startup_tab: z\ntabs:\n  - name: a\n    command: x\n", -1, "startup_tab"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.Decode(tc.yaml)
			require.NoError(t, err)

			err = config.Validate(cfg)

			fieldErr, ok := errors.AsType[*config.FieldError](err)
			require.True(t, ok, "want *FieldError, got %v", err)

			assert.Equal(t, tc.tab, fieldErr.Tab)
			assert.Equal(t, tc.field, fieldErr.Field)
		})
	}
}

func TestMarshalOmitsEmptyAndRoundTrips(t *testing.T) {
	cfg := &config.Config{
		Title:    "Dev",
		FontSize: 14,
		Tabs: []config.TabConfig{
			{Name: "api", Command: "go run .", RunOnStartup: true, StartupDelay: config.Duration{Duration: 2 * time.Second}},
			{Name: "web", Command: "npm run dev", Profiles: []string{"work"}, ShellArgs: []string{"-l", "-i"}},
		},
	}

	data, err := config.Marshal(cfg)
	require.NoError(t, err)

	want := `font_size: 14
title: Dev

tabs:
  - name: api
    command: go run .
    run_on_startup: true
    startup_delay: 2s

  - name: web
    command: npm run dev
    shell_args:
      - -l
      - -i
    profiles:
      - work
`
	assert.Equal(t, want, string(data))

	back, err := config.Decode(string(data))
	require.NoError(t, err)

	assert.Equal(t, cfg, back)
}

func TestMarshalMultilineCommand(t *testing.T) {
	cfg := &config.Config{Tabs: []config.TabConfig{{Name: "a", Command: "echo 1\n\necho 2\n"}}}

	data, err := config.Marshal(cfg)
	require.NoError(t, err)

	back, err := config.Decode(string(data))
	require.NoError(t, err)

	assert.Equal(t, cfg.Tabs[0].Command, back.Tabs[0].Command)
}

func TestHasComments(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want bool
	}{
		{"none", "tabs:\n  - name: a\n    command: x\n", false},
		{"hash in value", "tabs:\n  - name: a\n    command: echo '#x'\n", false},
		{"head", "# top\ntabs:\n  - name: a\n    command: x\n", true},
		{"line", "tabs:\n  - name: a # api\n    command: x\n", true},
		{"nested head", "tabs:\n  # first\n  - name: a\n    command: x\n", true},
		{"invalid yaml", "tabs: [\n", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, config.HasComments(tc.yaml))
		})
	}
}

func TestSave(t *testing.T) {
	t.Run("writes new file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "devtabs.yaml")
		cfg := &config.Config{Tabs: []config.TabConfig{{Name: "a", Command: "x"}}}

		require.NoError(t, config.Save(path, cfg))

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		assert.Equal(t, "tabs:\n  - name: a\n    command: x\n", string(data))
	})

	t.Run("keeps file mode and leaves no temp files", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "devtabs.yaml")

		require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))

		require.NoError(t, config.Save(path, &config.Config{Tabs: []config.TabConfig{{Name: "a", Command: "x"}}}))

		info, err := os.Stat(path)
		require.NoError(t, err)

		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)

		assert.Len(t, entries, 1)
	})
}
