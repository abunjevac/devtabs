package ui

import (
	"context"
	"os"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/abunjevac/devtabs/internal/config"
)

// Run starts the GTK application. Blocks until the window is closed.
func Run(ctx context.Context, cfg *config.Config, configDir string) {
	app := gtk.NewApplication("io.github.abunjevac.devtabs", gio.ApplicationNonUnique)

	app.ConnectActivate(func() {
		w := newWindow(ctx, app, cfg, configDir)

		w.Present()
	})

	os.Exit(app.Run(os.Args[:1]))
}
