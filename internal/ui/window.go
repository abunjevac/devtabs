package ui

import (
	"fmt"

	"github.com/abunjevac/devtabs/internal/config"
	"github.com/abunjevac/devtabs/internal/vte"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type toolbarButtons struct {
	run, stop *gtk.Button
}

func newWindow(app *gtk.Application, cfg *config.Config) *gtk.ApplicationWindow { //nolint:funlen,gocognit,gocyclo
	win := gtk.NewApplicationWindow(app)

	win.SetTitle("devtabs")
	win.SetDefaultSize(1200, 800)

	notebook := gtk.NewNotebook()
	notebook.SetTabPos(gtk.PosBottom)
	notebook.SetShowBorder(false)
	notebook.SetVExpand(true)

	fontFamily := cfg.Font
	fontSize := cfg.FontSize

	var tabs []*tab

	for i := range cfg.Tabs {
		t := newTab(&cfg.Tabs[i])

		vteTerm, vteWidget := vte.NewTerminal()

		t.terminal = vteTerm
		t.widget = vteWidget

		vte.SetFont(vteTerm, fontFamily, fontSize)

		notebook.AppendPage(vteWidget, t.labelWidget(i))

		vte.SpawnAsync(vteTerm, cfg.Tabs[i].WorkingDir, cfg.Tabs[i].Shell, cfg.Tabs[i].ShellArgs, t.onSpawnDone)

		vte.ConnectChildExited(vteTerm, func(_ int) {
			t.die()
		})

		tabs = append(tabs, t)
	}

	applyFont := func() {
		for _, t := range tabs {
			vte.SetFont(t.terminal, fontFamily, fontSize)
		}
	}

	toolbar, btns := buildToolbar(notebook, tabs)

	minus := gtk.NewButtonWithLabel("−")
	plus := gtk.NewButtonWithLabel("+")
	fontSep := gtk.NewSeparator(gtk.OrientationVertical)

	minus.SetFocusOnClick(false)
	plus.SetFocusOnClick(false)

	minus.ConnectClicked(func() {
		if fontSize > 6 {
			fontSize--
			applyFont()
		}
	})

	plus.ConnectClicked(func() {
		if fontSize < 72 {
			fontSize++
			applyFont()
		}
	})

	toolbar.Append(fontSep)
	toolbar.Append(minus)
	toolbar.Append(plus)

	vbox := gtk.NewBox(gtk.OrientationVertical, 0)

	vbox.Append(toolbar)
	vbox.Append(notebook)
	win.SetChild(vbox)

	for _, t := range tabs {
		t.onStateChange = func(_ tabState) {
			idx := notebook.CurrentPage()

			if idx >= 0 && idx < len(tabs) && tabs[idx] == t {
				updateButtonSensitivity(btns, tabs, idx)
			}
		}
	}

	notebook.ConnectSwitchPage(func(_ gtk.Widgetter, pageNum uint) {
		updateButtonSensitivity(btns, tabs, int(pageNum))

		if int(pageNum) < len(tabs) {
			tabs[pageNum].focus()
		}
	})

	updateButtonSensitivity(btns, tabs, 0)

	if cfg.StartupTab != "" {
		for i, tc := range cfg.Tabs {
			if tc.Name == cfg.StartupTab {
				notebook.SetCurrentPage(i)

				break
			}
		}
	}

	win.ConnectCloseRequest(func() (ok bool) {
		for _, t := range tabs {
			t.close()
		}

		return false
	})

	win.ConnectMap(func() {
		idx := notebook.CurrentPage()

		if idx >= 0 && idx < len(tabs) {
			tabs[idx].focus()
		}
	})

	keyCtrl := gtk.NewEventControllerKey()

	keyCtrl.SetPropagationPhase(gtk.PhaseCapture)
	win.AddController(keyCtrl)

	keyCtrl.ConnectKeyPressed(func(key, _ uint, state gdk.ModifierType) (ok bool) {
		switch {
		case state&gdk.AltMask != 0 && key >= uint('1') && key <= uint('9'):
			n := int(key - uint('1'))

			if n < len(tabs) {
				notebook.SetCurrentPage(n)

				return true
			}

		case state&gdk.AltMask != 0 && key == uint('r'):
			idx := notebook.CurrentPage()

			if idx >= 0 && idx < len(tabs) && tabs[idx].getState() == stateIdle {
				tabs[idx].runCommand()
			}

			return true

		case state&gdk.AltMask != 0 && key == uint('a'):
			for _, t := range tabs {
				if t.getState() == stateIdle {
					t.runCommand()
				}
			}

			return true

		case state&gdk.AltMask != 0 && key == uint('x'):
			for _, t := range tabs {
				if t.getState() == stateRunning {
					vte.FeedChild(t.terminal, "\x03")
				}
			}

			return true

		case state&gdk.ControlMask != 0 && key == uint('q'):
			win.Close()

			return true
		}

		return false
	})

	return win
}

func buildToolbar(notebook *gtk.Notebook, tabs []*tab) (*gtk.Box, toolbarButtons) { //nolint:cyclop
	run := shortcutButton("Run", "alt+r")
	stop := gtk.NewButtonWithLabel("Stop")
	runAll := shortcutButton("Run All", "alt+a")
	stopAll := shortcutButton("Stop All", "alt+x")
	sep := gtk.NewSeparator(gtk.OrientationVertical)

	for _, b := range []*gtk.Button{run, stop, runAll, stopAll} {
		b.SetFocusOnClick(false)
	}

	run.ConnectClicked(func() {
		idx := notebook.CurrentPage()

		if idx >= 0 && idx < len(tabs) && tabs[idx].getState() == stateIdle {
			tabs[idx].runCommand()
		}
	})

	stop.ConnectClicked(func() {
		idx := notebook.CurrentPage()

		if idx >= 0 && idx < len(tabs) && tabs[idx].getState() == stateRunning {
			vte.FeedChild(tabs[idx].terminal, "\x03")
		}
	})

	runAll.ConnectClicked(func() {
		for _, t := range tabs {
			if t.getState() == stateIdle {
				t.runCommand()
			}
		}
	})

	stopAll.ConnectClicked(func() {
		for _, t := range tabs {
			if t.getState() == stateRunning {
				vte.FeedChild(t.terminal, "\x03")
			}
		}
	})

	box := gtk.NewBox(gtk.OrientationHorizontal, 4)

	box.SetMarginStart(6)
	box.SetMarginEnd(6)
	box.SetMarginTop(4)
	box.SetMarginBottom(4)
	box.Append(run)
	box.Append(stop)
	box.Append(sep)
	box.Append(runAll)
	box.Append(stopAll)

	return box, toolbarButtons{run: run, stop: stop}
}

func updateButtonSensitivity(buttons toolbarButtons, tabs []*tab, idx int) {
	if idx < 0 || idx >= len(tabs) {
		buttons.run.SetSensitive(false)
		buttons.stop.SetSensitive(false)

		return
	}

	s := tabs[idx].getState()

	buttons.run.SetSensitive(s == stateIdle)
	buttons.stop.SetSensitive(s == stateRunning)
}

// shortcutButton creates a button whose label shows the keyboard hint in small dimmed text.
func shortcutButton(label, hint string) *gtk.Button {
	btn := gtk.NewButton()
	lbl := gtk.NewLabel("")

	lbl.SetMarkup(fmt.Sprintf("%s <span size='small' alpha='50%%'>%s</span>", label, hint))
	btn.SetChild(lbl)

	return btn
}
