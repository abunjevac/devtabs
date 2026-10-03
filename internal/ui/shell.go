package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func openTerminal(ctx context.Context, dir, terminal string, extraEnv []string) error {
	if terminal != "" {
		if err := startInDir(ctx, extraEnv, terminal, dir); err != nil {
			return fmt.Errorf("open-terminal: %w", err)
		}

		return nil
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

	var startErrs []error

	for _, c := range candidates {
		path, err := exec.LookPath(c.bin)
		if err != nil {
			continue
		}

		if err := command(ctx, extraEnv, path, c.args...).Start(); err != nil {
			startErrs = append(startErrs, fmt.Errorf("%s: %w", c.bin, err))

			continue
		}

		return nil
	}

	if len(startErrs) > 0 {
		return fmt.Errorf("open-terminal: %w", errors.Join(startErrs...))
	}

	return errors.New("open-terminal: no supported terminal emulator found")
}

func openFileManager(ctx context.Context, dir, fileManager string, extraEnv []string) error {
	if fileManager == "" {
		if err := command(ctx, extraEnv, "xdg-open", dir).Start(); err != nil {
			return fmt.Errorf("open-file-manager: %w", err)
		}

		return nil
	}

	if err := startInDir(ctx, extraEnv, fileManager, dir, "."); err != nil {
		return fmt.Errorf("open-file-manager: %w", err)
	}

	return nil
}

func openEditor(ctx context.Context, dir, editor string, extraEnv []string) error {
	if err := startInDir(ctx, extraEnv, editor, dir, "."); err != nil {
		return fmt.Errorf("open-editor: %w", err)
	}

	return nil
}

func startInDir(ctx context.Context, extraEnv []string, name, dir string, args ...string) error {
	cmd := command(ctx, extraEnv, name, args...)

	cmd.Dir = dir

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start-command: %w", err)
	}

	return nil
}

// command builds a command whose environment adds extraEnv (KEY=value) to devtabs' own.
func command(ctx context.Context, extraEnv []string, name string, args ...string) *exec.Cmd {
	//nolint:gosec // runs commands from the user's own config and restarts devtabs itself
	cmd := exec.CommandContext(ctx, name, args...)

	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}

	return cmd
}
