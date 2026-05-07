package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abunjevac/devtabs/internal/config"
)

func TestFlexBoolUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
		ok    bool
	}{
		{name: "true", input: "true", want: true, ok: true},
		{name: "True", input: "True", want: true, ok: true},
		{name: "TRUE", input: "TRUE", want: true, ok: true},
		{name: "yes", input: "yes", want: true, ok: true},
		{name: "YES", input: "YES", want: true, ok: true},
		{name: "on", input: "on", want: true, ok: true},
		{name: "ON", input: "ON", want: true, ok: true},
		{name: "1", input: "1", want: true, ok: true},
		{name: "false", input: "false", want: false, ok: true},
		{name: "False", input: "False", want: false, ok: true},
		{name: "no", input: "no", want: false, ok: true},
		{name: "off", input: "off", want: false, ok: true},
		{name: "0", input: "0", want: false, ok: true},
		{name: "invalid maybe", input: "maybe", want: false, ok: false},
		{name: "null (yaml null → zero false)", input: "", want: false, ok: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			yaml := "tabs:\n  - name: t\n    command: x\n    run_on_startup: " + tc.input
			cfg, err := config.LoadBytes([]byte(yaml))

			if tc.ok {
				require.NoError(t, err)
				assert.Equal(t, tc.want, bool(cfg.Tabs[0].RunOnStartup))
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestDurationUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Duration
		ok    bool
	}{
		{name: "5s", input: "5s", want: 5 * time.Second, ok: true},
		{name: "500ms", input: "500ms", want: 500 * time.Millisecond, ok: true},
		{name: "1m30s", input: "1m30s", want: 90 * time.Second, ok: true},
		{name: "0s", input: "0s", want: 0, ok: true},
		{name: "invalid string", input: "notaduration", ok: false},
		{name: "negative", input: "-1s", ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			yaml := "tabs:\n  - name: t\n    command: x\n    startup_delay: " + tc.input
			cfg, err := config.LoadBytes([]byte(yaml))

			if tc.ok {
				require.NoError(t, err)
				assert.Equal(t, tc.want, cfg.Tabs[0].StartupDelay.Duration)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	t.Run("missing name", func(t *testing.T) {
		_, err := config.LoadBytes([]byte("tabs:\n  - command: foo\n"))
		assert.Error(t, err)
	})

	t.Run("missing command", func(t *testing.T) {
		_, err := config.LoadBytes([]byte("tabs:\n  - name: foo\n"))
		assert.Error(t, err)
	})

	t.Run("duplicate names", func(t *testing.T) {
		_, err := config.LoadBytes([]byte("tabs:\n  - name: a\n    command: x\n  - name: a\n    command: y\n"))
		assert.Error(t, err)
	})

	t.Run("startup_tab not found", func(t *testing.T) {
		_, err := config.LoadBytes([]byte("startup_tab: missing\ntabs:\n  - name: a\n    command: x\n"))
		assert.Error(t, err)
	})

	t.Run("empty tabs", func(t *testing.T) {
		_, err := config.LoadBytes([]byte("tabs: []\n"))
		assert.Error(t, err)
	})

	t.Run("unknown field rejected", func(t *testing.T) {
		_, err := config.LoadBytes([]byte("tabs:\n  - name: a\n    command: x\n    unknown_field: oops\n"))
		assert.Error(t, err)
	})

	t.Run("valid minimal", func(t *testing.T) {
		cfg, err := config.LoadBytes([]byte("tabs:\n  - name: api\n    command: go run .\n"))
		require.NoError(t, err)

		require.Len(t, cfg.Tabs, 1)
		assert.Equal(t, "/bin/bash", cfg.Tabs[0].Shell)
		assert.Equal(t, []string{"-l"}, cfg.Tabs[0].ShellArgs)
	})

	t.Run("valid with startup_tab", func(t *testing.T) {
		cfg, err := config.LoadBytes([]byte("startup_tab: api\ntabs:\n  - name: api\n    command: go run .\n"))
		require.NoError(t, err)

		assert.Equal(t, "api", cfg.StartupTab)
	})
}
