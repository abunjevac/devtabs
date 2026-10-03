package ui

import (
	"context"
	"os"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/abunjevac/devtabs/internal/config"
	"github.com/abunjevac/devtabs/internal/editor"
)

// Run starts the GTK application. Blocks until the window is closed.
func Run(ctx context.Context, cfg *config.Config, configDir, configPath string) {
	childEnv := detachThemeOverride()
	app := newApplication()

	app.ConnectActivate(func() {
		w := newWindow(ctx, app, cfg, configDir, configPath, childEnv)

		w.Present()
	})

	os.Exit(app.Run(os.Args[:1]))
}

// RunEditor starts the application with only the config editor, for when the
// config cannot be loaded. notice explains why. Blocks until the editor is closed.
func RunEditor(ctx context.Context, configPath, notice string) {
	childEnv := detachThemeOverride()
	app := newApplication()

	app.ConnectActivate(func() {
		editor.Open(ctx, app, nil, editor.Options{
			Path:    configPath,
			Notice:  notice,
			Restart: func() { restartProcess(ctx, childEnv) },
		})
	})

	os.Exit(app.Run(os.Args[:1]))
}

func newApplication() *gtk.Application {
	app := gtk.NewApplication("io.github.abunjevac.devtabs", gio.ApplicationNonUnique)

	app.ConnectStartup(initAdwaita)

	return app
}

// initAdwaita starts libadwaita. libadwaita rejects GtkSettings'
// gtk-application-prefer-dark-theme (KDE writes it to settings.ini), so the
// setting is cleared first and its value carried over as the color scheme.
func initAdwaita() {
	settings := gtk.SettingsGetDefault()
	preferDark, _ := settings.ObjectProperty("gtk-application-prefer-dark-theme").(bool)

	if preferDark {
		settings.SetObjectProperty("gtk-application-prefer-dark-theme", false)
	}

	adw.Init()

	if preferDark {
		adw.StyleManagerGetDefault().SetColorScheme(adw.ColorSchemePreferDark)
	}
}

// detachThemeOverride removes GTK_THEME from this process, since a theme forced
// that way overrides libadwaita's styling. The returned environment entries restore
// it for launched processes, so they keep the user's theme.
func detachThemeOverride() []string {
	theme, ok := os.LookupEnv("GTK_THEME")
	if !ok {
		return nil
	}

	if err := os.Unsetenv("GTK_THEME"); err != nil {
		return nil
	}

	return []string{"GTK_THEME=" + theme}
}
