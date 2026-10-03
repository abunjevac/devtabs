package config_test

import (
	"testing"

	"github.com/abunjevac/devtabs/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrollbackLinesDefault(t *testing.T) {
	cfg, err := config.LoadFromString("tabs:\n  - name: a\n    command: x\n")
	require.NoError(t, err)

	assert.Equal(t, 2000, cfg.ScrollbackLines)
	assert.Equal(t, 2000, cfg.Tabs[0].ScrollbackLines)
}

func TestScrollbackLinesGlobalAppliesToTabs(t *testing.T) {
	cfg, err := config.LoadFromString("scrollback_lines: 5000\ntabs:\n  - name: a\n    command: x\n")
	require.NoError(t, err)

	assert.Equal(t, 5000, cfg.Tabs[0].ScrollbackLines)
}

func TestScrollbackLinesTabOverride(t *testing.T) {
	cfg, err := config.LoadFromString(
		"scrollback_lines: 5000\ntabs:\n  - name: a\n    command: x\n    scrollback_lines: -1\n  - name: b\n    command: y\n",
	)
	require.NoError(t, err)

	assert.Equal(t, -1, cfg.Tabs[0].ScrollbackLines)
	assert.Equal(t, 5000, cfg.Tabs[1].ScrollbackLines)
}

func TestScrollbackLinesRejectsBelowMinusOne(t *testing.T) {
	_, err := config.LoadFromString("scrollback_lines: -2\ntabs:\n  - name: a\n    command: x\n")
	require.Error(t, err)

	_, err = config.LoadFromString("tabs:\n  - name: a\n    command: x\n    scrollback_lines: -5\n")
	require.Error(t, err)
}
