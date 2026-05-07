package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/abunjevac/devtabs/internal/config"
	"github.com/abunjevac/devtabs/internal/ui"
	"github.com/abunjevac/devtabs/internal/version"
)

func main() {
	app := &cli.Command{
		Name:    "devtabs",
		Version: version.Version,
		Usage:   "terminal tab launcher from YAML config",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "./devtabs.yaml",
				Usage:   "path to YAML config file",
			},
			&cli.StringFlag{
				Name:    "root",
				Aliases: []string{"r"},
				Usage:   "base directory for resolving relative working_dir paths (default: config file directory)",
			},
		},
		Action: run,
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "devtabs: %v\n", err)

		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	cfgPath := cmd.String("config")
	root := cmd.String("root")

	if root == "" {
		abs, err := filepath.Abs(cfgPath)
		if err != nil {
			return fmt.Errorf("resolve-config-path: %w", err)
		}

		root = filepath.Dir(abs)
	}

	cfg, err := config.Load(cfgPath, root)
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	ui.Run(ctx, cfg)

	return nil
}
