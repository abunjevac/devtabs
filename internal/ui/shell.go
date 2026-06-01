package ui

import (
	"context"
	"os/exec"
)

func openTerminal(dir string) {
	type entry struct {
		bin  string
		args []string
	}

	candidates := []entry{
		{"x-terminal-emulator", []string{"--working-directory=" + dir}},
		{"gnome-terminal", []string{"--working-directory=" + dir}},
		{"xfce4-terminal", []string{"--working-directory=" + dir}},
		{"konsole", []string{"--workdir", dir}},
	}

	for _, c := range candidates {
		path, err := exec.LookPath(c.bin)
		if err != nil {
			continue
		}

		_ = exec.CommandContext(context.Background(), path, c.args...).Start()

		return
	}
}

func openFileManager(dir string) {
	_ = exec.CommandContext(context.Background(), "xdg-open", dir).Start()
}
