package ui

import (
	"context"
	"os"

	"github.com/abunjevac/devtabs/internal/config"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// Run starts the GTK application. Blocks until the window is closed.
func Run(ctx context.Context, cfg *config.Config) {
	app := gtk.NewApplication("io.github.abunjevac.devtabs", gio.ApplicationNonUnique)

	app.ConnectActivate(func() {
		w := newWindow(ctx, app, cfg)

		w.Present()
	})

	os.Exit(app.Run(os.Args[:1]))
}
