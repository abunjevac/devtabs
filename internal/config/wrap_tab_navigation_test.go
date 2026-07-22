package config_test

import (
	"testing"

	"github.com/abunjevac/devtabs/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapTabNavigation(t *testing.T) {
	cfg, err := config.LoadFromString("wrap_tab_navigation: true\ntabs:\n  - name: a\n    command: x\n")
	require.NoError(t, err)

	assert.True(t, cfg.WrapTabNavigation)
}

func TestWrapTabNavigationDefaultsToFalse(t *testing.T) {
	cfg, err := config.LoadFromString("tabs:\n  - name: a\n    command: x\n")
	require.NoError(t, err)

	assert.False(t, cfg.WrapTabNavigation)
}
