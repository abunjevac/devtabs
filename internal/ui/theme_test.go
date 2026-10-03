package ui

import (
	"context"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetachThemeOverride(t *testing.T) {
	t.Run("moves GTK_THEME to child environment", func(t *testing.T) {
		t.Setenv("GTK_THEME", "Breeze")

		env := detachThemeOverride()

		_, set := os.LookupEnv("GTK_THEME")

		assert.False(t, set)
		assert.Equal(t, []string{"GTK_THEME=Breeze"}, env)
	})

	t.Run("no override", func(t *testing.T) {
		// Setenv registers restoring the original value; then remove it for the test
		t.Setenv("GTK_THEME", "")

		require.NoError(t, os.Unsetenv("GTK_THEME"))

		assert.Nil(t, detachThemeOverride())
	})
}

func TestCommandEnv(t *testing.T) {
	assert.Nil(t, command(context.Background(), nil, "true").Env)

	cmd := command(context.Background(), []string{"GTK_THEME=Breeze"}, "true")

	assert.True(t, slices.Contains(cmd.Env, "GTK_THEME=Breeze"))
	assert.Greater(t, len(cmd.Env), 1)
}
