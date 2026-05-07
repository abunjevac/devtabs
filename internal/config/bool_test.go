package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoolUnmarshal(t *testing.T) {
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
			cfg, err := LoadBytes([]byte(yaml))

			if tc.ok {
				require.NoError(t, err)
				assert.Equal(t, tc.want, bool(cfg.Tabs[0].RunOnStartup))
			} else {
				assert.Error(t, err)
			}
		})
	}
}
