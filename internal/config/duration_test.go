package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			cfg, err := LoadFromString(yaml)

			if tc.ok {
				require.NoError(t, err)
				assert.Equal(t, tc.want, cfg.Tabs[0].StartupDelay.Duration)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
