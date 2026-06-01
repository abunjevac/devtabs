package ui

import (
	"context"
	"os/exec"
)

func openTerminal(ctx context.Context, dir, terminal string) {
	if terminal != "" {
		startInDir(ctx, terminal, dir)

		return
	}

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

		_ = exec.CommandContext(ctx, path, c.args...).Start()

		return
	}
}

func openFileManager(ctx context.Context, dir, fileManager string) {
	if fileManager == "" {
		_ = exec.CommandContext(ctx, "xdg-open", dir).Start()

		return
	}

	startInDir(ctx, fileManager, dir, ".")
}

func openEditor(ctx context.Context, dir, editor string) {
	startInDir(ctx, editor, dir, ".")
}

func startInDir(ctx context.Context, name, dir string, args ...string) {
	cmd := exec.CommandContext(ctx, name, args...)

	cmd.Dir = dir

	_ = cmd.Start()
}
