package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/abunjevac/devtabs/internal/config"
	"github.com/abunjevac/devtabs/internal/vte"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type toolbarButtons struct {
	run, stop *gtk.Button
}

// appWindow holds all GTK4 objects and mutable UI state for the main window.
type appWindow struct {
	win      *gtk.ApplicationWindow
	notebook *gtk.Notebook
	tabs     []*tab
	buttons  toolbarButtons

	fontFamily  string
	fontSize    float64
	configDir   string
	terminal    string
	fileManager string
	editor      string
}

func newWindow(ctx context.Context, app *gtk.Application, cfg *config.Config, configDir string) *gtk.ApplicationWindow {
	w := &appWindow{
		fontFamily:  cfg.Font,
		fontSize:    cfg.FontSize,
		configDir:   configDir,
		terminal:    cfg.Terminal,
		fileManager: cfg.FileManager,
		editor:      cfg.Editor,
	}

	w.win = gtk.NewApplicationWindow(app)

	w.win.SetTitle(cfg.Title)
	w.win.SetDefaultSize(cfg.WindowWidth, cfg.WindowHeight)
	w.win.SetIconName("utilities-terminal")

	w.notebook = gtk.NewNotebook()

	w.notebook.SetTabPos(gtk.PosBottom)
	w.notebook.SetShowBorder(false)
	w.notebook.SetVExpand(true)

	w.buildTabs(cfg)

	toolbar := w.buildToolbar(ctx)

	vbox := gtk.NewBox(gtk.OrientationVertical, 0)

	vbox.Append(toolbar)
	vbox.Append(w.notebook)
	w.win.SetChild(vbox)

	w.connectCallbacks(ctx, cfg)

	return w.win
}

func (w *appWindow) buildTabs(cfg *config.Config) {
	w.tabs = make([]*tab, 0, len(cfg.Tabs))

	for i := range cfg.Tabs {
		t := newTab(&cfg.Tabs[i])

		vteTerm, vteWidget := vte.NewTerminal()

		t.terminal = vteTerm
		t.widget = vteWidget

		vte.SetFont(vteTerm, w.fontFamily, w.fontSize)
		vte.SpawnAsync(vteTerm, cfg.Tabs[i].WorkingDir, cfg.Tabs[i].Shell, cfg.Tabs[i].ShellArgs, t.onSpawnDone)

		w.notebook.AppendPage(vteWidget, t.labelWidget(i))

		vte.ConnectChildExited(vteTerm, func(_ int) {
			t.die()
		})

		w.tabs = append(w.tabs, t)
	}
}

func (w *appWindow) buildToolbar(ctx context.Context) *gtk.Box {
	run := shortcutButton("media-playback-start", "Run", "alt+r")
	stop := shortcutButton("media-playback-stop", "Stop", "alt+s")
	runAll := shortcutButton("system-run", "Run All", "alt+a")
	stopAll := shortcutButton("process-stop", "Stop All", "alt+x")

	for _, b := range []*gtk.Button{run, stop, runAll, stopAll} {
		b.SetFocusOnClick(false)
	}

	run.ConnectClicked(w.runCurrent)
	stop.ConnectClicked(w.stopCurrent)
	runAll.ConnectClicked(w.runAll)
	stopAll.ConnectClicked(w.stopAll)

	minus := shortcutButton("zoom-out", "Font −", "ctrl+-")
	plus := shortcutButton("zoom-in", "Font +", "ctrl++")

	for _, b := range []*gtk.Button{minus, plus} {
		b.SetFocusOnClick(false)
	}

	minus.ConnectClicked(w.decreaseFont)
	plus.ConnectClicked(w.increaseFont)

	w.buttons = toolbarButtons{run: run, stop: stop}

	spacer := gtk.NewBox(gtk.OrientationHorizontal, 0)
	spacer.SetHExpand(true)

	menuBtn := w.buildMenuButton(ctx)

	box := gtk.NewBox(gtk.OrientationHorizontal, 4)

	box.SetMarginStart(6)
	box.SetMarginEnd(6)
	box.SetMarginTop(4)
	box.SetMarginBottom(4)
	box.Append(run)
	box.Append(stop)
	box.Append(gtk.NewSeparator(gtk.OrientationVertical))
	box.Append(runAll)
	box.Append(stopAll)
	box.Append(gtk.NewSeparator(gtk.OrientationVertical))
	box.Append(minus)
	box.Append(plus)
	box.Append(gtk.NewSeparator(gtk.OrientationVertical))
	box.Append(w.buildDirButtons(ctx))
	box.Append(spacer)
	box.Append(menuBtn)

	return box
}

func (w *appWindow) buildDirButtons(ctx context.Context) *gtk.Box {
	openTerm := shortcutButton("utilities-terminal", "Terminal", "alt+t")
	openFiles := shortcutButton("system-file-manager", "Files", "alt+f")
	openEdit := shortcutButton("accessories-text-editor", "Editor", "alt+e")

	for _, b := range []*gtk.Button{openTerm, openFiles, openEdit} {
		b.SetFocusOnClick(false)
	}

	openTerm.ConnectClicked(func() { w.openCurrentTerminal(ctx) })
	openFiles.ConnectClicked(func() { w.openCurrentFileManager(ctx) })
	openEdit.ConnectClicked(func() { w.openCurrentEditor(ctx) })

	box := gtk.NewBox(gtk.OrientationHorizontal, 4)

	box.Append(openTerm)
	box.Append(openFiles)
	box.Append(openEdit)

	return box
}

func (w *appWindow) connectCallbacks(ctx context.Context, cfg *config.Config) { //nolint:cyclop
	for _, t := range w.tabs {
		t.onStateChange = func(_ tabState) {
			idx := w.notebook.CurrentPage()

			if idx >= 0 && idx < len(w.tabs) && w.tabs[idx] == t {
				w.updateButtonSensitivity(idx)
			}
		}
	}

	w.notebook.ConnectSwitchPage(func(_ gtk.Widgetter, pageNum uint) {
		w.updateButtonSensitivity(int(pageNum))

		if int(pageNum) < len(w.tabs) {
			w.tabs[pageNum].focus()
		}
	})

	w.updateButtonSensitivity(0)

	if cfg.StartupTab != "" {
		for i, tc := range cfg.Tabs {
			if tc.Name == cfg.StartupTab {
				w.notebook.SetCurrentPage(i)

				break
			}
		}
	}

	w.win.ConnectCloseRequest(func() bool {
		for _, t := range w.tabs {
			t.close()
		}

		return false
	})

	w.win.ConnectMap(func() {
		idx := w.notebook.CurrentPage()

		if idx >= 0 && idx < len(w.tabs) {
			w.tabs[idx].focus()
		}
	})

	w.installKeyController(ctx)
}

func (w *appWindow) installKeyController(ctx context.Context) {
	keyCtrl := gtk.NewEventControllerKey()

	keyCtrl.SetPropagationPhase(gtk.PhaseCapture)
	w.win.AddController(keyCtrl)

	keyCtrl.ConnectKeyPressed(func(key, keycode uint, state gdk.ModifierType) bool {
		return w.onKeyPressed(ctx, key, keycode, state)
	})
}

func (w *appWindow) onKeyPressed(ctx context.Context, key, _ uint, state gdk.ModifierType) bool { //nolint:cyclop
	altPressed := state&gdk.AltMask != 0
	ctrlPressed := state&gdk.ControlMask != 0

	switch {
	case altPressed && key >= uint('1') && key <= uint('9'):
		n := int(key - uint('1'))

		if n < len(w.tabs) {
			w.notebook.SetCurrentPage(n)

			return true
		}

	case altPressed && key == uint('s'):
		w.stopCurrent()

	case altPressed && key == uint('r'):
		w.runCurrent()

	case altPressed && key == uint('a'):
		w.runAll()

	case altPressed && key == uint('x'):
		w.stopAll()

	case altPressed && key == uint('t'):
		w.openCurrentTerminal(ctx)

	case altPressed && key == uint('f'):
		w.openCurrentFileManager(ctx)

	case altPressed && key == uint('e'):
		w.openCurrentEditor(ctx)

	case ctrlPressed && (key == gdk.KEY_plus || key == gdk.KEY_equal || key == gdk.KEY_KP_Add):
		w.increaseFont()

	case ctrlPressed && (key == gdk.KEY_minus || key == gdk.KEY_KP_Subtract):
		w.decreaseFont()

	case ctrlPressed && key == uint('q'):
		w.win.Close()

	default:
		return false
	}

	return true
}

func (w *appWindow) runCurrent() {
	idx := w.notebook.CurrentPage()

	if idx >= 0 && idx < len(w.tabs) && w.tabs[idx].getState() == stateIdle {
		w.tabs[idx].runCommand()
	}
}

func (w *appWindow) stopCurrent() {
	idx := w.notebook.CurrentPage()

	if idx >= 0 && idx < len(w.tabs) && w.tabs[idx].getState() == stateRunning {
		vte.FeedChild(w.tabs[idx].terminal, "\x03")
	}
}

func (w *appWindow) runAll() {
	for _, t := range w.tabs {
		if t.getState() == stateIdle {
			t.runCommand()
		}
	}
}

func (w *appWindow) stopAll() {
	for _, t := range w.tabs {
		if t.getState() == stateRunning {
			vte.FeedChild(t.terminal, "\x03")
		}
	}
}

func (w *appWindow) increaseFont() {
	if w.fontSize < 72 {
		w.fontSize++

		w.applyFont()
	}
}

func (w *appWindow) decreaseFont() {
	if w.fontSize > 6 {
		w.fontSize--

		w.applyFont()
	}
}

func (w *appWindow) applyFont() {
	for _, t := range w.tabs {
		vte.SetFont(t.terminal, w.fontFamily, w.fontSize)
	}
}

func (w *appWindow) openCurrentTerminal(ctx context.Context) {
	if dir, ok := w.currentTabDir(); ok {
		openTerminal(ctx, dir, w.terminal)
	}
}

func (w *appWindow) openCurrentFileManager(ctx context.Context) {
	if dir, ok := w.currentTabDir(); ok {
		openFileManager(ctx, dir, w.fileManager)
	}
}

func (w *appWindow) openCurrentEditor(ctx context.Context) {
	if dir, ok := w.currentTabDir(); ok {
		openEditor(ctx, dir, w.editor)
	}
}

func (w *appWindow) currentTabDir() (string, bool) {
	idx := w.notebook.CurrentPage()

	if idx < 0 || idx >= len(w.tabs) {
		return "", false
	}

	return w.tabs[idx].cfg.WorkingDir, true
}

func (w *appWindow) updateButtonSensitivity(idx int) {
	if idx < 0 || idx >= len(w.tabs) {
		w.buttons.run.SetSensitive(false)
		w.buttons.stop.SetSensitive(false)

		return
	}

	s := w.tabs[idx].getState()

	w.buttons.run.SetSensitive(s == stateIdle)
	w.buttons.stop.SetSensitive(s == stateRunning)
}

// restartProcess starts a fresh copy of the binary with the same arguments and exits.
func restartProcess(ctx context.Context) {
	exe, err := os.Executable()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "devtabs: restart: %v\n", err)

		return
	}

	//nolint:gosec // restarting the same binary with the original arguments is intentional
	cmd := exec.CommandContext(ctx, exe, os.Args[1:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "devtabs: restart: %v\n", err)

		return
	}

	os.Exit(0)
}

func (w *appWindow) buildMenuButton(ctx context.Context) *gtk.MenuButton {
	popover := gtk.NewPopover()

	openDirItems := w.buildOpenDirMenuItems(ctx, popover)

	restartBtn := menuItem("view-refresh", "Restart")

	restartBtn.ConnectClicked(func() {
		popover.Popdown()

		for _, t := range w.tabs {
			t.close()
		}

		restartProcess(ctx)
	})

	quitBtn := menuItem("application-exit", "Quit")

	quitBtn.ConnectClicked(func() {
		popover.Popdown()
		w.win.Close()
	})

	popoverBox := gtk.NewBox(gtk.OrientationVertical, 2)

	popoverBox.SetMarginTop(4)
	popoverBox.SetMarginBottom(4)
	popoverBox.SetMarginStart(4)
	popoverBox.SetMarginEnd(4)

	for _, item := range openDirItems {
		popoverBox.Append(item)
	}

	popoverBox.Append(gtk.NewSeparator(gtk.OrientationHorizontal))
	popoverBox.Append(restartBtn)
	popoverBox.Append(quitBtn)

	popover.SetChild(popoverBox)

	menuBtn := gtk.NewMenuButton()

	menuBtn.SetIconName("open-menu-symbolic")
	menuBtn.SetPopover(popover)
	menuBtn.SetFocusOnClick(false)

	return menuBtn
}

func (w *appWindow) buildOpenDirMenuItems(ctx context.Context, popover *gtk.Popover) []*gtk.Button {
	openTerm := menuItem("utilities-terminal", "Open Terminal Here")

	openTerm.ConnectClicked(func() {
		popover.Popdown()

		openTerminal(ctx, w.configDir, w.terminal)
	})

	openFiles := menuItem("system-file-manager", "Open Files Here")

	openFiles.ConnectClicked(func() {
		popover.Popdown()

		openFileManager(ctx, w.configDir, w.fileManager)
	})

	openEditorBtn := menuItem("accessories-text-editor", "Open Editor Here")

	openEditorBtn.ConnectClicked(func() {
		popover.Popdown()

		openEditor(ctx, w.configDir, w.editor)
	})

	return []*gtk.Button{openTerm, openFiles, openEditorBtn}
}

// menuItem creates a frameless, left-aligned button with an icon and label for use in a popover menu.
func menuItem(iconName, label string) *gtk.Button {
	img := gtk.NewImageFromIconName(iconName)

	img.SetPixelSize(16)

	lbl := gtk.NewLabel(label)

	lbl.SetHAlign(gtk.AlignStart)
	lbl.SetHExpand(true)

	row := gtk.NewBox(gtk.OrientationHorizontal, 8)

	row.Append(img)
	row.Append(lbl)

	btn := gtk.NewButton()

	btn.SetChild(row)
	btn.SetHasFrame(false)
	btn.SetFocusOnClick(false)

	return btn
}

// shortcutButton creates a button with an icon, a label, and an optional keyboard hint.
// Pass an empty hint to omit the hint text.
func shortcutButton(iconName, label, hint string) *gtk.Button {
	btn := gtk.NewButton()
	img := gtk.NewImageFromIconName(iconName)
	lbl := gtk.NewLabel("")

	img.SetPixelSize(16)

	if hint != "" {
		lbl.SetMarkup(fmt.Sprintf("%s <span size='small' alpha='50%%'>%s</span>", label, hint))
	} else {
		lbl.SetText(label)
	}

	box := gtk.NewBox(gtk.OrientationHorizontal, 4)

	box.Append(img)
	box.Append(lbl)
	btn.SetChild(box)

	return btn
}
