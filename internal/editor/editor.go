// Package editor provides a libadwaita window for editing a devtabs config file.
package editor

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/abunjevac/devtabs/internal/config"
)

var errNegativeDuration = errors.New("duration must not be negative")

// Options configures the editor window.
type Options struct {
	// absolute path of the config file to edit
	Path string
	// shown in a banner when the file loads, e.g. why devtabs could not start
	Notice string
	// called after a successful Save & Restart
	Restart func()
}

// invalidInput is text the user typed that could not be parsed.
type invalidInput struct {
	tab   *config.TabConfig
	title string
	text  string
}

type editor struct {
	document

	opts    Options
	win     *adw.ApplicationWindow
	toasts  *adw.ToastOverlay
	split   *adw.NavigationSplitView
	view    *adw.ToolbarView
	title   *adw.WindowTitle
	sidebar *gtk.ListBox
	rows    map[*config.TabConfig]*sidebarRow

	// page currently shown; nil is the General page
	current *config.TabConfig
	// editable widgets of the current page, by YAML field name
	fields  map[string]gtk.Widgetter
	invalid map[string]invalidInput
	dirty   bool
}

// Open shows the config editor. parent may be nil when no main window exists.
func Open(ctx context.Context, app *gtk.Application, parent *gtk.Window, opts Options) {
	e := &editor{
		document: loadDocument(opts.Path, opts.Notice),
		opts:     opts,
		rows:     make(map[*config.TabConfig]*sidebarRow),
		fields:   make(map[string]gtk.Widgetter),
		invalid:  make(map[string]invalidInput),
	}

	e.build(ctx, app)

	if parent != nil {
		e.win.SetTransientFor(parent)
	}

	e.rebuildSidebar(nil)

	e.win.Present()
}

func (e *editor) build(ctx context.Context, app *gtk.Application) {
	e.win = adw.NewApplicationWindow(app)

	e.win.SetTitle("Edit Config")
	e.win.SetDefaultSize(980, 760)
	e.win.SetSizeRequest(720, 480)

	e.split = adw.NewNavigationSplitView()

	e.split.SetMinSidebarWidth(220)
	e.split.SetSidebar(adw.NewNavigationPage(e.buildSidebarView(ctx), "Config"))
	e.split.SetContent(adw.NewNavigationPage(e.buildContentView(ctx), "Settings"))

	e.toasts = adw.NewToastOverlay()

	e.toasts.SetChild(e.split)

	e.win.SetContent(e.toasts)

	e.win.ConnectCloseRequest(func() bool { return e.onCloseRequest(ctx) })
}

func (e *editor) buildSidebarView(ctx context.Context) *adw.ToolbarView {
	add := gtk.NewButtonFromIconName("list-add-symbolic")

	add.SetTooltipText("Add tab")
	add.ConnectClicked(e.addTab)

	header := adw.NewHeaderBar()

	header.PackStart(add)

	e.sidebar = e.buildSidebar(ctx)

	scroller := gtk.NewScrolledWindow()

	scroller.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scroller.SetChild(e.sidebar)

	view := adw.NewToolbarView()

	view.AddTopBar(header)
	view.SetContent(scroller)

	return view
}

func (e *editor) buildContentView(ctx context.Context) *adw.ToolbarView {
	save := gtk.NewButtonWithLabel("Save")

	save.ConnectClicked(func() { e.save(nil) })

	restart := gtk.NewButtonWithLabel("Save & Restart")

	restart.AddCSSClass("suggested-action")
	restart.ConnectClicked(func() { e.save(e.opts.Restart) })

	cancel := gtk.NewButtonWithLabel("Cancel")

	cancel.ConnectClicked(func() { e.cancel(ctx) })

	e.title = adw.NewWindowTitle("", "")

	header := adw.NewHeaderBar()

	header.SetTitleWidget(e.title)
	header.PackEnd(restart)
	header.PackEnd(save)
	header.PackEnd(cancel)

	e.view = adw.NewToolbarView()

	e.view.AddTopBar(header)

	if e.notice != "" {
		banner := adw.NewBanner(e.notice)

		banner.SetButtonLabel("Dismiss")
		banner.SetRevealed(true)
		banner.ConnectButtonClicked(func() { banner.SetRevealed(false) })

		e.view.AddTopBar(banner)
	}

	return e.view
}

// showPage replaces the form with the General page (nil) or the given tab.
func (e *editor) showPage(ctx context.Context, t *config.TabConfig) {
	e.current = t

	clear(e.fields)

	if t == nil {
		e.view.SetContent(e.generalPage())
	} else {
		e.view.SetContent(e.tabPage(ctx, t))
	}

	e.updateTitle()
}

func (e *editor) updateTitle() {
	title := "General"

	if e.current != nil {
		title = cmp.Or(e.current.Name, "Unnamed")
	}

	subtitle := e.opts.Path

	// the path is ellipsized at the end, so the marker goes first
	if e.dirty {
		subtitle = "Modified • " + subtitle
	}

	e.title.SetTitle(title)
	e.title.SetSubtitle(subtitle)
}

func (e *editor) markDirty() {
	if e.dirty {
		return
	}

	e.dirty = true

	e.updateTitle()
}

// fieldKey identifies a field on the current page across page rebuilds.
func (e *editor) fieldKey(field string) string {
	return fmt.Sprintf("%p/%s", e.current, field)
}

func (e *editor) addTab() {
	t := &config.TabConfig{Name: newTabName(e.tabs)}

	e.tabs = append(e.tabs, t)

	e.markDirty()

	e.rebuildSidebar(t)
}

func (e *editor) deleteTab(t *config.TabConfig) {
	i := slices.Index(e.tabs, t)

	if i < 0 || len(e.tabs) == 1 {
		return
	}

	e.tabs = slices.Delete(e.tabs, i, i+1)

	if e.general.StartupTab == t.Name {
		e.general.StartupTab = ""
	}

	for key, in := range e.invalid {
		if in.tab == t {
			delete(e.invalid, key)
		}
	}

	e.markDirty()

	e.rebuildSidebar(e.tabs[max(i-1, 0)])
}

func (e *editor) moveTab(from, to int) {
	if from == to || from < 0 || to < 0 || from >= len(e.tabs) || to >= len(e.tabs) {
		return
	}

	t := e.tabs[from]

	e.tabs = slices.Insert(slices.Delete(e.tabs, from, from+1), to, t)

	e.markDirty()

	e.rebuildSidebar(t)
}

// renameTab keeps startup_tab pointing at the tab when its name changes.
func (e *editor) renameTab(t *config.TabConfig, name string) {
	if t.Name != "" && e.general.StartupTab == t.Name {
		e.general.StartupTab = name
	}

	t.Name = name

	e.refreshSidebarRow(t)

	e.updateTitle()
}

// save validates and writes the config, then runs then on success.
func (e *editor) save(then func()) {
	if e.reportInvalidInput() {
		return
	}

	cfg := e.config()

	if err := config.Validate(cfg); err != nil {
		e.reportValidationError(err)

		return
	}

	if !e.hadComments {
		e.write(cfg, then)

		return
	}

	dialog := adw.NewAlertDialog("Remove Comments?", "The config file contains comments. Saving rewrites the file and removes them.")

	dialog.AddResponse("cancel", "Cancel")
	dialog.AddResponse("save", "Save Anyway")
	dialog.SetResponseAppearance("save", adw.ResponseDestructive)
	dialog.SetDefaultResponse("cancel")
	dialog.SetCloseResponse("cancel")

	dialog.ConnectResponse(func(response string) {
		if response == "save" {
			e.write(cfg, then)
		}
	})

	dialog.Present(e.win)
}

func (e *editor) write(cfg *config.Config, then func()) {
	if err := config.Save(e.opts.Path, cfg); err != nil {
		e.toast(fmt.Sprintf("Save failed: %v", err))

		return
	}

	e.hadComments = false
	e.dirty = false

	e.updateTitle()

	e.toast("Saved")

	if then != nil {
		then()
	}
}

// reportInvalidInput jumps to the first field holding unparsable text.
func (e *editor) reportInvalidInput() bool {
	if len(e.invalid) == 0 {
		return false
	}

	pages := append([]*config.TabConfig{nil}, e.tabs...)

	for _, page := range pages {
		for _, in := range e.invalid {
			if in.tab != page {
				continue
			}

			e.rebuildSidebar(page)

			e.toast(fmt.Sprintf("%s: %q is not a valid value", in.title, strings.TrimSpace(in.text)))

			return true
		}
	}

	return true
}

func (e *editor) reportValidationError(err error) {
	fieldErr, ok := errors.AsType[*config.FieldError](err)
	if !ok {
		e.toast(err.Error())

		return
	}

	var page *config.TabConfig

	if fieldErr.Tab >= 0 && fieldErr.Tab < len(e.tabs) {
		page = e.tabs[fieldErr.Tab]
	}

	e.rebuildSidebar(page)

	if w, ok := e.fields[fieldErr.Field]; ok {
		widget := gtk.BaseWidget(w)

		widget.AddCSSClass("error")
		widget.GrabFocus()
	}

	e.toast(fieldErr.Msg)
}

func (e *editor) toast(msg string) {
	e.toasts.AddToast(adw.NewToast(msg))
}

// cancel closes the editor without saving, confirming first when there are changes.
func (e *editor) cancel(ctx context.Context) {
	if !e.dirty {
		e.win.Close()

		return
	}

	dialog := adw.NewAlertDialog("Discard Changes?", "Unsaved changes to the config will be lost.")

	dialog.AddResponse("keep", "Keep Editing")
	dialog.AddResponse("discard", "Discard")
	dialog.SetResponseAppearance("discard", adw.ResponseDestructive)
	dialog.SetDefaultResponse("keep")
	dialog.SetCloseResponse("keep")

	dialog.Choose(ctx, e.win, func(res gio.AsyncResulter) {
		if dialog.ChooseFinish(res) == "discard" {
			e.dirty = false

			e.win.Close()
		}
	})
}

func (e *editor) onCloseRequest(ctx context.Context) bool {
	if !e.dirty {
		return false
	}

	dialog := adw.NewAlertDialog("Save Changes?", "The config has unsaved changes.")

	dialog.AddResponse("cancel", "Cancel")
	dialog.AddResponse("discard", "Discard")
	dialog.AddResponse("save", "Save")
	dialog.SetResponseAppearance("discard", adw.ResponseDestructive)
	dialog.SetResponseAppearance("save", adw.ResponseSuggested)
	dialog.SetDefaultResponse("save")
	dialog.SetCloseResponse("cancel")

	dialog.Choose(ctx, e.win, func(res gio.AsyncResulter) {
		switch dialog.ChooseFinish(res) {
		case "discard":
			e.dirty = false

			e.win.Close()
		case "save":
			e.save(func() { e.win.Close() })
		}
	})

	return true
}
