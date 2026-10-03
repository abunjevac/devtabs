package editor

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/abunjevac/devtabs/internal/config"
	"github.com/abunjevac/devtabs/internal/version"
)

const noStartupTab = "First tab"

func (e *editor) generalPage() *adw.PreferencesPage {
	g := &e.general
	page := adw.NewPreferencesPage()

	window := group("Window", "")

	window.Add(e.textRow("title", "Title", "Window title bar text", "devtabs "+version.Version, g.Title, func(s string) { g.Title = s }))
	window.Add(e.intRow("window_width", "Width", "Initial window width in pixels", "1200", g.WindowWidth, func(n int) { g.WindowWidth = n }))
	window.Add(e.intRow("window_height", "Height", "Initial window height in pixels", "800", g.WindowHeight, func(n int) { g.WindowHeight = n }))

	terminal := group("Terminal", "")

	terminal.Add(e.textRow("font", "Font", "Terminal font family", "Monospace", g.Font, func(s string) { g.Font = s }))
	terminal.Add(e.parsedRow("font_size", "Font size", "In points; adjust at runtime with Ctrl++ and Ctrl+-", "12", formatFloat(g.FontSize), func(s string) error {
		return parseFloat(s, &g.FontSize)
	}))
	terminal.Add(e.intRow("scrollback_lines", "Scrollback lines", "Default for all tabs; -1 for unlimited", "2000", g.ScrollbackLines, func(n int) { g.ScrollbackLines = n }))

	navigation := group("Navigation", "")

	navigation.Add(e.startupTabRow())
	navigation.Add(e.switchRow("Wrap tab navigation", "Alt+Right on the last tab moves to the first", g.WrapTabNavigation, func(v bool) { g.WrapTabNavigation = v }))

	actions := group("Directory Actions", "Commands used by the Terminal, Files and Editor buttons")

	actions.Add(e.textRow("terminal", "Terminal", "Terminal emulator to open", "auto-detect", g.Terminal, func(s string) { g.Terminal = s }))
	actions.Add(e.textRow("file_manager", "File manager", "File manager to open", "xdg-open", g.FileManager, func(s string) { g.FileManager = s }))
	actions.Add(e.textRow("editor", "Editor", "Editor command to open", "zed", g.Editor, func(s string) { g.Editor = s }))

	page.Add(window)
	page.Add(terminal)
	page.Add(navigation)
	page.Add(actions)

	return page
}

func (e *editor) startupTabRow() *adw.ComboRow {
	names := []string{noStartupTab}

	for _, t := range e.tabs {
		if t.Name != "" && !slices.Contains(names, t.Name) {
			names = append(names, t.Name)
		}
	}

	// keep an unknown value visible so the user can see and fix it
	if s := e.general.StartupTab; s != "" && !slices.Contains(names[1:], s) {
		names = append(names, s)
	}

	row := adw.NewComboRow()

	row.SetTitle("Startup tab")
	row.SetSubtitle("Tab focused when devtabs launches")
	row.SetModel(gtk.NewStringList(names))

	if i := slices.Index(names[1:], e.general.StartupTab); i >= 0 && e.general.StartupTab != "" {
		row.SetSelected(uint(i + 1))
	}

	row.NotifyProperty("selected", func() {
		i := int(row.Selected())

		e.general.StartupTab = ""

		if i > 0 && i < len(names) {
			e.general.StartupTab = names[i]
		}

		e.markDirty()
	})

	e.fields["startup_tab"] = row

	return row
}

func (e *editor) tabPage(ctx context.Context, t *config.TabConfig) *adw.PreferencesPage {
	page := adw.NewPreferencesPage()

	basics := group("Tab", "Alt command runs with Alt+T instead of Alt+R")

	basics.Add(e.entryRow("name", "Name", t.Name, func(s string) { e.renameTab(t, s) }))
	basics.Add(e.entryRow("command", "Command", t.Command, func(s string) { t.Command = s; e.refreshSidebarRow(t) }))
	basics.Add(e.entryRow("alt_command", "Alt command", t.AltCommand, func(s string) { t.AltCommand = s }))

	dir := group("Working Directory", "Relative paths resolve against the config file's directory; empty uses that directory")

	dir.Add(e.workingDirRow(ctx, t))

	startup := group("Startup", "")

	startup.Add(e.switchRow("Run on startup", "Run the command when devtabs launches", bool(t.RunOnStartup), func(v bool) { t.RunOnStartup = config.Bool(v) }))
	startup.Add(e.parsedRow("startup_delay", "Startup delay", "Wait before running, e.g. 2s or 500ms", "0s", formatDuration(t.StartupDelay), func(s string) error {
		return parseDuration(s, &t.StartupDelay)
	}))

	shell := group("Shell", "")

	shell.Add(e.textRow("shell", "Shell", "Shell binary started in the tab", "/bin/zsh", t.Shell, func(s string) { t.Shell = s }))
	shell.Add(e.textRow("shell_args", "Shell arguments", "Comma-separated", "-l", formatList(t.ShellArgs), func(s string) { t.ShellArgs = parseList(s) }))

	advanced := group("Advanced", "")

	advanced.Add(e.textRow("profiles", "Profiles", "Comma-separated; shown only with a matching --profile. Empty means always shown", "", formatList(t.Profiles), func(s string) { t.Profiles = parseList(s) }))
	advanced.Add(e.intRow("scrollback_lines", "Scrollback lines", "Overrides the global value; -1 for unlimited", e.scrollbackDefault(), t.ScrollbackLines, func(n int) { t.ScrollbackLines = n }))

	page.Add(basics)
	page.Add(dir)
	page.Add(startup)
	page.Add(shell)
	page.Add(advanced)
	page.Add(e.deleteGroup(t))

	return page
}

func (e *editor) workingDirRow(ctx context.Context, t *config.TabConfig) *adw.EntryRow {
	row := e.entryRow("working_dir", "Working directory", t.WorkingDir, func(s string) { t.WorkingDir = s })
	browse := gtk.NewButtonFromIconName("folder-open-symbolic")

	browse.SetTooltipText("Choose folder")
	browse.SetVAlign(gtk.AlignCenter)
	browse.AddCSSClass("flat")

	browse.ConnectClicked(func() {
		dialog := gtk.NewFileDialog()

		dialog.SetTitle("Working Directory")
		dialog.SelectFolder(ctx, &e.win.Window, func(res gio.AsyncResulter) {
			folder, err := dialog.SelectFolderFinish(res)
			if err != nil || folder == nil {
				return
			}

			row.SetText(folder.Path())
		})
	})

	row.AddSuffix(browse)

	return row
}

func (e *editor) deleteGroup(t *config.TabConfig) *adw.PreferencesGroup {
	button := gtk.NewButtonWithLabel("Delete Tab")

	button.AddCSSClass("destructive-action")
	button.AddCSSClass("pill")
	button.SetHAlign(gtk.AlignCenter)
	button.SetSensitive(len(e.tabs) > 1)

	button.ConnectClicked(func() {
		e.deleteTab(t)
	})

	g := adw.NewPreferencesGroup()

	g.Add(button)

	return g
}

func (e *editor) scrollbackDefault() string {
	if e.general.ScrollbackLines != 0 {
		return strconv.Itoa(e.general.ScrollbackLines)
	}

	return "2000"
}

func formatFloat(f float64) string {
	if f == 0 {
		return ""
	}

	return strconv.FormatFloat(f, 'f', -1, 64)
}

func parseFloat(s string, dst *float64) error {
	if s == "" {
		*dst = 0

		return nil
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("parse-float: %w", err)
	}

	*dst = f

	return nil
}

func formatDuration(d config.Duration) string {
	if d.Duration == 0 {
		return ""
	}

	return d.String()
}

func parseDuration(s string, dst *config.Duration) error {
	if s == "" {
		*dst = config.Duration{}

		return nil
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("parse-duration: %w", err)
	}

	if d < 0 {
		return errNegativeDuration
	}

	dst.Duration = d

	return nil
}
