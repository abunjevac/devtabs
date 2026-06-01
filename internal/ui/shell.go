package ui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

func openTerminal(ctx context.Context, dir, terminal string) error {
	if terminal != "" {
		if err := startInDir(ctx, terminal, dir); err != nil {
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

		if err := exec.CommandContext(ctx, path, c.args...).Start(); err != nil {
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

func openFileManager(ctx context.Context, dir, fileManager string) error {
	if fileManager == "" {
		if err := exec.CommandContext(ctx, "xdg-open", dir).Start(); err != nil {
			return fmt.Errorf("open-file-manager: %w", err)
		}

		return nil
	}

	if err := startInDir(ctx, fileManager, dir, "."); err != nil {
		return fmt.Errorf("open-file-manager: %w", err)
	}

	return nil
}

func openEditor(ctx context.Context, dir, editor string) error {
	if err := startInDir(ctx, editor, dir, "."); err != nil {
		return fmt.Errorf("open-editor: %w", err)
	}

	return nil
}

func startInDir(ctx context.Context, name, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)

	cmd.Dir = dir

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start-command: %w", err)
	}

	return nil
}
